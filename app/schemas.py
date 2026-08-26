"""中心服务的 Pydantic 请求 / 响应模型。

注意：任何输出模型都不包含 node_key 明文或封存密文。
"""

from __future__ import annotations

import datetime as dt

from pydantic import BaseModel, Field


class RegisterRequest(BaseModel):
    """节点注册请求。node_key 不再由客户端提交，由服务端自动生成。"""

    name: str = Field(min_length=1, max_length=128)
    address: str = Field(min_length=1, max_length=255)
    port: int = Field(ge=1, le=65535)
    protocol: str = Field(default="http")
    metadata: dict | None = None


class NodeOut(BaseModel):
    id: int
    name: str
    address: str
    port: int
    protocol: str
    key_fingerprint: str
    enabled: bool
    metadata: dict | None = None
    created_at: dt.datetime
    updated_at: dt.datetime
    last_seen_at: dt.datetime | None = None
    last_status: str
    last_latency_ms: float | None = None
    last_error: str | None = None


class NodeListOut(BaseModel):
    nodes: list[NodeOut]


class RegisterOut(NodeOut):
    """注册成功的响应。

    ``node_key`` 是一次性凭据：只在本次注册响应中明文返回一次，之后任何
    GET 列表/详情、错误信息、日志都不会再包含它，也无法通过 API 再次查看。
    请立即复制保存，并注入到对应的节点 agent（NODE_KEY 环境变量 / --node-key）。
    """

    node_key: str = Field(description="一次性凭据，仅在本次响应中出现，请立即保存")


class AutoRegisterRequest(RegisterRequest):
    """节点 agent 启动时自助注册所用的请求体。

    ``existing_node_key`` 可选：agent 把上次持久化在 node.ini 里的 key 带上，
    服务端只有在它确实属于同一个 address+port 的当前节点时才会复用（不轮换）；
    否则一律当作没带 key 处理，生成新 key。
    """

    existing_node_key: str | None = Field(
        default=None, description="上次注册得到的 node_key，用于尝试复用，不保证一定被采用"
    )


class AutoRegisterOut(RegisterOut):
    reused: bool = Field(description="true 表示复用了 existing_node_key（未轮换），false 表示生成了新 key")


class KeyedRequest(BaseModel):
    """当节点未保存密钥封存副本时，管理员需要在请求体里显式提供 node_key。"""

    node_key: str | None = None


class SpeedtestRequest(KeyedRequest):
    ping_count: int | None = Field(default=None, ge=1, le=50)
    download_bytes: int | None = Field(default=None, ge=1)
    upload_bytes: int | None = Field(default=None, ge=1)


class PingResult(BaseModel):
    count: int
    min_ms: float
    avg_ms: float
    max_ms: float
    jitter_ms: float


class TransferResult(BaseModel):
    bytes: int
    duration_ms: float
    mbps: float


class SpeedtestResult(BaseModel):
    node_id: int
    ping: PingResult | None = None
    download: TransferResult | None = None
    upload: TransferResult | None = None
    error: str | None = None


class HealthResult(BaseModel):
    node_id: int
    status: str
    latency_ms: float | None = None
    error: str | None = None
