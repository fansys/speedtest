# LibreSpeed 风格测速服务

用 Go 实现的自建分布式测速服务：一个中心 Web 服务（`cmd/web`，节点注册 / 管理 / 代理与发起测速 / 现代 Web 界面）
+ 可独立部署的节点 agent（`cmd/node`，ping / 流式 download / 流式 upload）。SQLite 存储节点信息
（`internal/sqlite3` 用 CGO 绑定系统 libsqlite3）。

## 架构

- **中心服务**（`cmd/web`）：Web 页面 + JSON API + 测速代理。负责节点注册、启用/禁用/删除、
  健康检查，提供浏览器实时流式测速代理，以及向指定节点发起 ping/download/upload 测速并汇总结果。使用 SQLite 存储节点信息。
- **节点 agent**（`cmd/node`）：独立进程，暴露 `/healthz`、`/ping`、流式 `/download`、
  流式 `/upload`，只接受携带正确 `node_key` 的请求（错误/缺失一律 401，支持 CORS 与 OPTIONS 预检）。不依赖中心服务的
  数据库，可以部署在任意机器/机房；上传/下载数据量都受 `max_test_bytes` 上限约束。

节点的密钥（`node_key`）：

- **由中心服务自动生成**，客户端调用 `POST /api/register` / `POST /api/register/auto` 时
  **不需要**（也不应该）提交 `node_key`；服务端生成高熵随机值。
- **只在注册成功的响应中明文返回**——之后任何 `GET /api/nodes`（列表/详情）、错误信息、日志
  都不会再包含它，也没有任何管理 API 能重新查看明文。
- 数据库里**只保存 sha256 哈希**（`node_key_hash`），绝不落地明文。
- 默认额外保存一份用 `SECRET_KEY` **加密封存**的副本（`node_key_sealed`），使中心服务可以自动
  代理或发起健康检查/测速，而不需要管理员每次手动输入 key。这是 `STORE_NODE_KEY_SEALED=true`（默认）。
  如果不希望中心持有任何可解密的副本，设置 `STORE_NODE_KEY_SEALED=false`，届时每次健康检查/
  测速都需要在请求体里显式提供 `node_key`（会先与库中哈希比对，确认无误后才使用，不会被保存）。
- **节点 agent 会自己完成注册**（见下文「节点自动注册」），并把拿到的 key 持久化在本地
  `node.ini` 里；重启时会带着 `node.ini` 里的旧 key 去尝试复用，只要还是同一个
  `address+port` 且旧 key 仍然有效，服务端就会**复用而不轮换**。只有在没有旧 key、节点不存在、
  已被删除、或旧 key 不匹配/已失效时，服务端才会生成新 key 并让旧 key 立即失效。
- 通过 `POST /api/register`（不带 `existing_node_key`）手动注册时，重复注册同一
  `address+port` 仍然一律生成新 key 并让旧 key 立即失效，这个行为没有变。

## 节点自动注册

节点 agent（`cmd/node`）启动 HTTP 服务前，会先完成注册，不需要人工 `curl` 或手动复制
`node_key`：

1. 读取本地 `node.ini`（路径由 `NODE_INI` 指定，默认 `./node.ini`，Docker 镜像里默认
   `/data/node.ini`），取出上次保存的 `node_key`（如果有）。
2. 调用中心服务的 `POST /api/register/auto`（地址由 `NODE_REGISTER_URL` 指定），带上
   `REGISTRATION_TOKEN`、节点信息，以及上一步读到的 `existing_node_key`。
   - 如果这个 key 仍然属于同一个 `address+port` 的当前节点，服务端**复用它**（不轮换）。
   - 否则（首次注册 / 节点被删除 / key 不匹配或已失效）服务端生成一个新 key。
3. 把服务端返回的 `node_id`、`node_key`、`name`、`address`、`port`、`protocol`、`updated_at`
   原子写回 `node.ini`（临时文件 + 替换，容器异常退出也不会写坏；文件权限尽量收紧到 `0600`）。
4. 用拿到的 key 启动 HTTP 服务。

网络暂时不可用时按指数退避重试（1、2、4、8、16... 秒，封顶 16 秒），最多重试
`NODE_REGISTER_RETRIES` 次（默认 10）；仍然失败则**直接退出**，不会带着未注册的状态启动。
日志里只打印 key 的指纹（哈希前 12 位），不会打印完整 `node_key` 或 `REGISTRATION_TOKEN`。

相关环境变量：

| 变量 | 说明 |
| --- | --- |
| `NODE_REGISTER_URL` | 注册地址，例如 `http://web:8080/api/register/auto`；只允许 http/https |
| `REGISTRATION_TOKEN` | 与中心服务一致的注册令牌 |
| `NODE_NAME` / `NODE_ADDRESS` / `NODE_PORT` / `NODE_PROTOCOL` | 注册时提交的节点信息 |
| `NODE_METADATA_JSON` | 可选，附带的任意 JSON 元数据字符串 |
| `NODE_INI` | node.ini 持久化路径 |
| `NODE_REGISTER_RETRIES` | 网络失败时的最大重试次数（默认 10） |
| `NODE_KEY` | 显式高级覆盖：非空时**跳过自动注册**，直接用这个 key 启动 |

如果没有设置 `NODE_REGISTER_URL`，则必须显式提供 `NODE_KEY`，否则启动失败——这就是原来的
「手动模式」，仍然完全可用。

### 查看已注册节点 / 处理失效 key

- 想看某个节点当前状态（在线/离线、上次测速结果等），在 Web 页面填入 Admin Token 后打开首页
  即可看到节点列表；`GET /api/nodes` 同样只返回哈希前 12 位的“指纹”，不会返回明文 key。
- 想确认某个节点当前用的是哪个 key（指纹），或者节点是否已经完成自动注册，直接查看它本地的
  `node.ini` 文件（Docker 场景下即 `node-data` volume 里的 `/data/node.ini`）。
- 如果怀疑某个节点的 key 已经失效（比如数据库被回滚、节点被误删后重建）：不需要手动处理，
  直接重启该节点容器/进程即可——它会带着 `node.ini` 里的旧 key 再次尝试注册，服务端发现不匹配
  后会自动生成新 key 并写回 `node.ini`。

## 快速开始（本地，不用 Docker）

需要 Go 1.22+（当前使用 Go 1.22 的 `http.ServeMux` 方法路由；CGO 需要系统装有 `gcc` 和
`libsqlite3-dev`，因为 `internal/sqlite3` 通过 CGO 动态链接系统 libsqlite3）。

```bash
cp .env.example .env
# 编辑 .env，填入 ADMIN_TOKEN / REGISTRATION_TOKEN / SECRET_KEY
# 生成随机令牌: openssl rand -base64 32

mkdir -p data

# 启动中心服务（从仓库根目录运行，读取 ./.env 和 ./static）
go run ./cmd/web
```

浏览器打开 `http://127.0.0.1:8080/`，在页面顶部填入 Admin Token / Registration Token（只保存在
浏览器本地）。

### 启动节点 agent（自动注册，推荐）

不需要手动 `curl` 或复制 key，节点 agent 启动前会自己调用 `POST /api/register/auto` 完成注册：

```bash
NODE_REGISTER_URL="http://127.0.0.1:8080/api/register/auto" \
REGISTRATION_TOKEN="<REGISTRATION_TOKEN>" \
NODE_NAME="上海-电信" \
NODE_ADDRESS="127.0.0.1" \
NODE_PORT=8081 \
NODE_INI="./node.ini" \
go run ./cmd/node
```

第一次启动会生成新 key 并写入 `./node.ini`；之后重启（比如换机器重新部署但 `node.ini` 还在）
会带着里面的旧 key 去尝试复用，只要地址/端口没变且旧 key 仍然有效，服务端就直接复用、不轮换。
详见上文「节点自动注册」一节。

### 启动节点 agent（手动指定 key，高级用法）

如果你想跳过自动注册、自己管理 key，可以先调用 `POST /api/register`（响应里的 `node_key`
**只会出现这一次**，请立即复制），再用它启动节点：

```bash
curl -X POST http://127.0.0.1:8080/api/register \
  -H "X-Registration-Token: <REGISTRATION_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"name": "上海-电信", "address": "127.0.0.1", "port": 8081, "protocol": "http"}'

NODE_KEY="<注册响应里的 node_key>" NODE_PORT=8081 NODE_NAME="上海-电信" go run ./cmd/node
```

Web 页面手动注册成功后同样会弹窗显示这个一次性 key（不会自动存入浏览器 localStorage，关闭弹窗前
请务必复制保存）。

### 管理 / 测速 API

所有 `/api/nodes/*` 接口都需要 `X-Admin-Token`（或 `Authorization: Bearer <token>`）请求头。

| 方法与路径 | 说明 |
| --- | --- |
| `GET /api/nodes` | 列出所有节点（不含 node_key，只含哈希前 12 位指纹） |
| `GET /api/nodes/{id}` | 查看单个节点 |
| `POST /api/nodes/{id}/enable` / `disable` | 启用 / 禁用节点 |
| `DELETE /api/nodes/{id}` | 删除节点 |
| `POST /api/nodes/{id}/health` | 对该节点发起一次后端健康检查 |
| `POST /api/nodes/{id}/speedtest` | 对该节点发起后端 ping + download + upload 完整测速 |
| `GET /api/nodes/{id}/ping` | 浏览器实时 Ping 代理 |
| `GET /api/nodes/{id}/download` | 浏览器实时流式下载测速代理 |
| `POST /api/nodes/{id}/upload` | 浏览器实时流式上传测速代理 |
| `GET /api/speedtest/ping` | Web 中心直连 Ping 测速 |
| `GET /api/speedtest/download` | Web 中心直连流式下载测速 |
| `POST /api/speedtest/upload` | Web 中心直连流式上传测速 |

`health` / `speedtest` 请求体是可选的 JSON：

```json
{
  "node_key": "仅当 STORE_NODE_KEY_SEALED=false 时必填",
  "ping_count": 6,
  "download_bytes": 33554432,
  "upload_bytes": 16777216
}
```

`speedtest` 响应示例：

```json
{
  "node_id": 1,
  "ping": {"count": 6, "min_ms": 3.1, "avg_ms": 4.0, "max_ms": 5.2, "jitter_ms": 2.1},
  "download": {"bytes": 33554432, "duration_ms": 812.3, "mbps": 330.5},
  "upload": {"bytes": 16777216, "duration_ms": 455.0, "mbps": 294.9},
  "error": null
}
```

节点不可达/拒绝密钥时，`error` 字段会给出可读原因，`ping/download/upload` 对应字段可能为空。

## 安全设计

- **鉴权**：`ADMIN_TOKEN` 保护管理/测速 API；`REGISTRATION_TOKEN` 保护注册 API；节点 agent 用
  `X-Node-Key`（或 `Authorization: Bearer`）校验，均使用恒定时间比较，且只从请求头读取（不接受
  query string，避免落入访问日志）。启动时若 `ADMIN_TOKEN` / `REGISTRATION_TOKEN` 缺失、过短
  （<16 字符）或两者相同，服务会拒绝启动。
- **node_key 由服务端生成、只存哈希、一次性返回**：见上文“架构”一节。任何列表/详情 API、错误
  信息、日志都不会包含 node_key 明文；节点列表只暴露哈希前 12 位的“指纹”，可用于人工核对但不可
  逆推。重复注册同一 `address+port` 会让旧 key 立即失效。
- **SSRF 防护**（`internal/netguard`）：节点地址只允许 `http`/`https`（可通过
  `ALLOWED_NODE_PROTOCOLS` 收紧）；拒绝带路径/查询串/认证信息的地址；端口拒绝一批明显非
  HTTP 的敏感端口（22/3306/6379/27017 等）；解析主机名得到的所有 A/AAAA 记录都会逐一检查。
  `ALLOW_PRIVATE_NODES`（默认 `true`）控制是否允许回环 / 私网 / 链路本地（含
  `169.254.169.254` 云元数据地址）/ 保留 / 多播地址——**默认允许是为了支持内网自建测速节点这一
  主要场景；如果你的部署环境需要防止内部人员用本服务当 SSRF 跳板攻击内网，生产环境务必把它设为
  `false`**。中心服务每次真正发起请求前都会重新校验一次目标地址，缩小 DNS rebinding 窗口。
- **令牌传输**：只走请求头，不接受 URL 参数；节点 agent 自动注册时 `existing_node_key` 也只放在
  JSON 请求体里，不会出现在 query string 或日志中；`REGISTRATION_TOKEN` 不写日志；`node.ini`
  文件权限尽量收紧到 `0600`；注册请求带超时；`NODE_REGISTER_URL` 只允许 `http`/`https`。
- **节点 agent 自身**：`/healthz`、`/ping`、`/download`、`/upload` 全部要求正确 `node_key`，缺失
  或错误一律 401 且不回显收到的值；上传/下载都有 `max_test_bytes` 大小上限，超出上传上限会
  中断并返回 413。

## 运行测试

```bash
gofmt -l .          # 格式检查，应无输出
go vet ./...
go test -v ./...
```

测试覆盖：
- 静态 UI 与 HTML / CSS / JS 资源可访问性
- 中心服务与节点 agent 健康检查接口
- 管理员与注册令牌鉴权（缺失、错误、Bearer、专用 Header）
- 自动注册首次生成、同节点旧 key 复用、key 错误自动重新生成轮换、跨节点不可复用
- 手动注册与旧 key 立即失效
- `node.ini` 0600 权限保护与原子写入
- SSRF netguard 校验（私网 IP、保留端口、非 HTTP 协议拦截）
- sealedbox AES-256-GCM 密文封存与解封
- 节点上传大小限制（超过上限返回 413）与下载截断保护
- 节点 agent CORS 与 OPTIONS 预检处理
- 浏览器代理实时流式测速与后端批量测速端到端验证

## CI/CD 与多平台发布

项目已集成 GitHub Actions 自动化工作流：

- **CI (`.github/workflows/ci.yml`)**：在 `push` / `pull_request` 时自动执行代码格式检查 (`gofmt`)、静态检查 (`go vet`)、单元测试 (`go test`) 及 Docker 构建验证。
- **Release (`.github/workflows/release.yml`)**：在推送版本标签（如 `v1.0.0`）或手动触发时自动：
  1. **多架构 Docker 镜像**：自动构建 `linux/amd64` 与 `linux/arm64` 镜像并推送到 GitHub Container Registry (GHCR)：
     - `ghcr.io/<owner>/librespeed-web:latest` & `v*`
     - `ghcr.io/<owner>/librespeed-node:latest` & `v*`
  2. **多平台二进制打包发布**：自动编译并归档到 GitHub Releases，附带 SHA256 校验和 (`checksums.txt`)：
     - **Linux** (`amd64`, `arm64`, `armv7`, `386`, `riscv64`)
     - **macOS / Darwin** (`amd64` Intel, `arm64` Apple Silicon)
     - **Windows** (`amd64`, `arm64`, `386`)
     - **FreeBSD** (`amd64`, `arm64`)

## Docker

中心服务和节点 agent 分别打包成两个独立镜像（`Dockerfile.web` / `Dockerfile.node`），各自可以
单独构建/运行。

### 分别构建 / 运行

```bash
# 中心服务
docker build -t librespeed-web -f Dockerfile.web .
docker run --rm -p 8080:8080 --env-file .env -v $(pwd)/data:/data librespeed-web

# 节点 agent（自动注册模式：容器启动前自己调用 web 服务完成注册）
docker build -t librespeed-node -f Dockerfile.node .
docker run --rm -p 8081:8081 \
  -e NODE_REGISTER_URL="http://<web 容器地址>:8080/api/register/auto" \
  -e REGISTRATION_TOKEN="<REGISTRATION_TOKEN>" \
  -e NODE_NAME="上海-电信" \
  -e NODE_ADDRESS="<中心服务能访问到本节点的地址>" \
  -e NODE_PORT=8081 \
  -v node-data:/data \
  librespeed-node
```

两个镜像都以非 root 用户运行（`webapp` / `nodeagent`），构建上下文只复制各自需要的
`go.mod`、`cmd/`、`internal/`（以及 web 镜像额外需要的 `static/`），不会把仓库里其它文件
打进镜像。节点 agent 镜像内置 `HEALTHCHECK`（`docker/node-healthcheck.sh`：自动注册模式下会从
`NODE_INI` 指向的 `node.ini` 里读取 key 再请求 `/healthz`）。

### 用 docker-compose 一起跑（推荐）

`docker-compose.yml` 里 `node` 服务 `depends_on` `web` 的健康检查（`condition:
service_healthy`），并且已经配好了自动注册所需的环境变量，**一条命令即可**：

```bash
cp .env.example .env
# 编辑 .env，填入 ADMIN_TOKEN / REGISTRATION_TOKEN / SECRET_KEY（不要把真实值提交到仓库）

docker compose up -d
```

启动顺序完全由 compose 自动处理：`web` 健康检查通过后 `node` 才会启动，`node` 容器起来后会
自己调用 `POST /api/register/auto` 完成注册，把凭据写入挂载的 `node-data` volume 里的
`/data/node.ini`（不会因为容器重建而丢失；下次启动会带着这个文件里的 key 尝试复用）。全程不需要
手动 `curl` 或复制 `node_key`。
