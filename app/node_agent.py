"""LibreSpeed 风格的独立节点 agent。

可单独启动，不依赖中心服务的数据库：

    python -m app.node_agent --host 0.0.0.0 --port 8081 --node-key <key>

所有测速接口都要求请求头携带正确的 node_key（``X-Node-Key`` 或
``Authorization: Bearer``），比较使用恒定时间算法，错误响应不回显收到的值。
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import sys
import time

from fastapi import FastAPI, Header, HTTPException, Request, Response, status
from fastapi.responses import StreamingResponse

from .node_register_client import NodeRegistrationError, register_with_retries
from .node_state import load_node_state, save_node_state
from .security import MIN_NODE_KEY_LENGTH, constant_time_equals, extract_bearer, key_fingerprint

DEFAULT_MAX_TEST_BYTES = 512 * 1024 * 1024
CHUNK_SIZE = 64 * 1024


def _random_chunk(size: int) -> bytes:
    return os.urandom(size)


def create_node_app(
    node_key: str,
    name: str = "node",
    *,
    max_test_bytes: int = DEFAULT_MAX_TEST_BYTES,
) -> FastAPI:
    if not node_key or len(node_key) < MIN_NODE_KEY_LENGTH:
        raise ValueError(f"node_key 长度至少 {MIN_NODE_KEY_LENGTH} 个字符")

    app = FastAPI(title=f"LibreSpeed Node Agent - {name}")
    app.state.node_key = node_key
    app.state.name = name
    app.state.max_test_bytes = max_test_bytes
    app.state.download_chunk = _random_chunk(CHUNK_SIZE)

    def _authenticate(x_node_key: str | None, authorization: str | None) -> None:
        supplied = x_node_key.strip() if x_node_key and x_node_key.strip() else extract_bearer(
            authorization
        )
        if not supplied or not constant_time_equals(supplied, app.state.node_key):
            raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="node_key 无效或缺失")

    @app.get("/healthz")
    async def healthz(
        x_node_key: str | None = Header(default=None, alias="X-Node-Key"),
        authorization: str | None = Header(default=None),
    ) -> dict:
        _authenticate(x_node_key, authorization)
        return {"status": "ok", "name": app.state.name}

    @app.get("/ping")
    async def ping(
        x_node_key: str | None = Header(default=None, alias="X-Node-Key"),
        authorization: str | None = Header(default=None),
    ) -> Response:
        _authenticate(x_node_key, authorization)
        return Response(status_code=status.HTTP_204_NO_CONTENT)

    @app.get("/download")
    async def download(
        bytes: int = 10_000_000,
        x_node_key: str | None = Header(default=None, alias="X-Node-Key"),
        authorization: str | None = Header(default=None),
    ) -> StreamingResponse:
        _authenticate(x_node_key, authorization)
        total = max(1, min(bytes, app.state.max_test_bytes))
        chunk = app.state.download_chunk

        def generator():
            sent = 0
            while sent < total:
                n = min(len(chunk), total - sent)
                yield chunk[:n]
                sent += n

        return StreamingResponse(
            generator(),
            media_type="application/octet-stream",
            headers={"Content-Length": str(total)},
        )

    @app.post("/upload")
    async def upload(
        request: Request,
        x_node_key: str | None = Header(default=None, alias="X-Node-Key"),
        authorization: str | None = Header(default=None),
    ) -> dict:
        _authenticate(x_node_key, authorization)
        start = time.perf_counter()
        total = 0
        async for chunk in request.stream():
            total += len(chunk)
            if total > app.state.max_test_bytes:
                raise HTTPException(status_code=413, detail="上传数据超过节点允许的上限")
        elapsed = max(time.perf_counter() - start, 1e-9)
        mbps = (total * 8 / 1_000_000) / elapsed
        return {"bytes": total, "duration_ms": elapsed * 1000, "mbps": mbps}

    return app


def _build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="LibreSpeed 风格节点 agent")
    parser.add_argument("--host", default=os.environ.get("NODE_HOST", "0.0.0.0"))
    parser.add_argument("--port", type=int, default=int(os.environ.get("NODE_PORT", "8081")))
    parser.add_argument("--name", default=os.environ.get("NODE_NAME", "node"))
    parser.add_argument("--log-level", default=os.environ.get("NODE_LOG_LEVEL", "info"))

    # 显式高级覆盖：设置了就直接使用这个 key 启动，跳过下面的自动注册。
    parser.add_argument("--node-key", default=os.environ.get("NODE_KEY", ""))

    # 自动注册相关配置。
    parser.add_argument(
        "--register-url",
        default=os.environ.get("NODE_REGISTER_URL") or os.environ.get("REGISTER_URL", ""),
    )
    parser.add_argument("--registration-token", default=os.environ.get("REGISTRATION_TOKEN", ""))
    parser.add_argument("--address", default=os.environ.get("NODE_ADDRESS", ""))
    parser.add_argument("--protocol", default=os.environ.get("NODE_PROTOCOL", "http"))
    parser.add_argument("--metadata-json", default=os.environ.get("NODE_METADATA_JSON", ""))
    parser.add_argument("--node-ini", default=os.environ.get("NODE_INI", "./node.ini"))
    parser.add_argument(
        "--register-retries", type=int, default=int(os.environ.get("NODE_REGISTER_RETRIES", "10"))
    )
    return parser


def resolve_node_key(args: argparse.Namespace) -> str:
    """按优先级解析出启动要用的 node_key：显式覆盖 > 自动注册 > 报错退出。

    自动注册模式下会先读取 node.ini 里上次保存的 key 作为 existing_node_key
    带给服务端尝试复用，成功/复用后都会把最新状态原子写回 node.ini。
    """
    node_key = (args.node_key or "").strip()
    if node_key:
        print(f"[node-agent] 使用显式提供的 NODE_KEY 启动（跳过自动注册），指纹={key_fingerprint(node_key)}")
        return node_key

    if not args.register_url:
        print(
            "错误: 必须通过 --node-key / NODE_KEY 显式提供密钥，或设置 NODE_REGISTER_URL 启用自动注册。\n"
            "手动生成 key: python -c \"import secrets;print(secrets.token_urlsafe(32))\"",
            file=sys.stderr,
        )
        raise SystemExit(1)

    if not args.registration_token:
        print("错误: 设置了 NODE_REGISTER_URL 时必须同时提供 REGISTRATION_TOKEN。", file=sys.stderr)
        raise SystemExit(1)
    if not args.address:
        print("错误: 自动注册模式需要 NODE_ADDRESS（中心服务据此访问本节点）。", file=sys.stderr)
        raise SystemExit(1)

    metadata: dict | None = None
    if args.metadata_json:
        try:
            metadata = json.loads(args.metadata_json)
        except ValueError as exc:
            print(f"错误: NODE_METADATA_JSON 不是合法 JSON: {exc}", file=sys.stderr)
            raise SystemExit(1) from exc

    state = load_node_state(args.node_ini) or {}
    existing_key = (state.get("node_key") or "").strip() or None

    try:
        result = register_with_retries(
            args.register_url,
            args.registration_token,
            name=args.name,
            address=args.address,
            port=args.port,
            protocol=args.protocol,
            metadata=metadata,
            existing_node_key=existing_key,
            retries=args.register_retries,
        )
    except NodeRegistrationError as exc:
        print(f"错误: 自动注册失败，节点未启动: {exc}", file=sys.stderr)
        raise SystemExit(1) from exc

    node_key = (result.get("node_key") or "").strip()
    if not node_key:
        print("错误: 注册响应中没有 node_key，节点未启动。", file=sys.stderr)
        raise SystemExit(1)

    save_node_state(
        args.node_ini,
        node_id=result.get("id"),
        node_key=node_key,
        name=args.name,
        address=args.address,
        port=args.port,
        protocol=args.protocol,
        updated_at=dt.datetime.now(dt.timezone.utc).isoformat(),
    )

    status_word = "复用了已有" if result.get("reused") else "生成了新的"
    print(
        f"[node-agent] 自动注册成功，{status_word} node_key，"
        f"node_id={result.get('id')} 指纹={key_fingerprint(node_key)}"
    )
    return node_key


def main(argv: list[str] | None = None) -> None:
    args = _build_parser().parse_args(argv)
    node_key = resolve_node_key(args)

    import uvicorn

    app = create_node_app(node_key, args.name)
    uvicorn.run(app, host=args.host, port=args.port, log_level=args.log_level)


if __name__ == "__main__":
    main()
