"""集中配置。

所有敏感值都只从环境变量 / .env 读取，绝不写入日志或 API 响应。
"""

from __future__ import annotations

from functools import lru_cache
from typing import Literal

from pydantic import Field, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

MIN_TOKEN_LENGTH = 16


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    # ---- 鉴权 ----
    admin_token: str = Field(default="", description="管理 API 令牌")
    registration_token: str = Field(default="", description="节点注册令牌")
    secret_key: str = Field(default="", description="用于封存 node_key 的主密钥")

    # ---- 服务 ----
    host: str = "127.0.0.1"
    port: int = 8080
    database_url: str = "sqlite:///./data/librespeed.db"
    log_level: str = "info"

    # ---- 节点访问 / SSRF 控制 ----
    allow_private_nodes: bool = True
    allowed_node_protocols: str = "http,https"
    node_connect_timeout: float = 5.0
    node_request_timeout: float = 30.0
    node_health_timeout: float = 5.0

    # ---- 测速参数 ----
    max_test_bytes: int = 512 * 1024 * 1024
    default_download_bytes: int = 32 * 1024 * 1024
    default_upload_bytes: int = 16 * 1024 * 1024
    stream_chunk_bytes: int = 64 * 1024
    ping_count: int = 6

    # ---- 节点 key 存储策略 ----
    # true  : 除 sha256 哈希外，额外保存一份用 SECRET_KEY 加密封存的 node_key，
    #         中心服务可自动做健康检查 / 测速。
    # false : 只保存哈希，任何对节点的访问都必须由调用方显式提供 node_key。
    store_node_key_sealed: bool = True

    @field_validator("allowed_node_protocols")
    @classmethod
    def _normalize_protocols(cls, value: str) -> str:
        parts = [p.strip().lower() for p in value.split(",") if p.strip()]
        allowed = {"http", "https"}
        bad = set(parts) - allowed
        if bad:
            raise ValueError(f"ALLOWED_NODE_PROTOCOLS 只支持 http/https，收到: {sorted(bad)}")
        if not parts:
            raise ValueError("ALLOWED_NODE_PROTOCOLS 不能为空")
        return ",".join(parts)

    @model_validator(mode="after")
    def _check_secrets(self) -> "Settings":
        for name, value in (
            ("ADMIN_TOKEN", self.admin_token),
            ("REGISTRATION_TOKEN", self.registration_token),
        ):
            if not value:
                raise ValueError(
                    f"{name} 未设置。请在 .env 中配置，可用 "
                    "`python -c \"import secrets;print(secrets.token_urlsafe(32))\"` 生成。"
                )
            if len(value) < MIN_TOKEN_LENGTH:
                raise ValueError(f"{name} 长度至少 {MIN_TOKEN_LENGTH} 个字符。")
        if self.admin_token == self.registration_token:
            raise ValueError("ADMIN_TOKEN 与 REGISTRATION_TOKEN 不能相同。")
        if self.store_node_key_sealed and len(self.secret_key) < MIN_TOKEN_LENGTH:
            raise ValueError(
                "启用 STORE_NODE_KEY_SEALED 时必须设置长度 >= "
                f"{MIN_TOKEN_LENGTH} 的 SECRET_KEY。"
            )
        return self

    @property
    def protocol_list(self) -> list[str]:
        return self.allowed_node_protocols.split(",")

    def secret_values(self) -> list[str]:
        """所有需要在日志中屏蔽的敏感值。"""
        return [v for v in (self.admin_token, self.registration_token, self.secret_key) if v]


ProtocolLiteral = Literal["http", "https"]


@lru_cache(maxsize=1)
def get_settings() -> Settings:
    return Settings()
