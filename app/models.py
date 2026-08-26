"""SQLAlchemy 2.0 ORM 模型。"""

from __future__ import annotations

import datetime as dt

from sqlalchemy import Boolean, DateTime, Float, Integer, String, Text, UniqueConstraint
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


def utcnow() -> dt.datetime:
    return dt.datetime.now(dt.timezone.utc)


class Base(DeclarativeBase):
    pass


class Node(Base):
    """一个测速节点。

    安全约定：
    - ``node_key_hash`` 是 node_key 的 sha256，用于校验，不可反推。
    - ``node_key_sealed`` 是可选的加密封存副本（依赖 SECRET_KEY），仅供中心服务
      主动访问节点时解封使用，任何 API 都不会返回它。
    - ``key_fingerprint`` 是哈希前 12 位，可安全展示，用于人工区分 key。
    """

    __tablename__ = "nodes"
    __table_args__ = (
        UniqueConstraint("address", "port", name="uq_nodes_address_port"),
        # SQLite 默认会在删除表中 id 最大的行后复用该 id（因为它只是取
        # max(rowid)+1）。这里显式要求 AUTOINCREMENT，让 SQLite 用
        # sqlite_sequence 记录历史最大 id，保证被删除节点的 id 永不复用，
        # 避免旧 node_key 被“捡漏”绑定到语义上全新的节点。
        {"sqlite_autoincrement": True},
    )

    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    name: Mapped[str] = mapped_column(String(128), nullable=False)
    address: Mapped[str] = mapped_column(String(255), nullable=False)
    port: Mapped[int] = mapped_column(Integer, nullable=False)
    protocol: Mapped[str] = mapped_column(String(8), nullable=False, default="http")

    node_key_hash: Mapped[str] = mapped_column(String(64), nullable=False, index=True)
    key_fingerprint: Mapped[str] = mapped_column(String(16), nullable=False)
    node_key_sealed: Mapped[str | None] = mapped_column(Text, nullable=True)

    node_metadata: Mapped[str | None] = mapped_column("metadata_json", Text, nullable=True)

    enabled: Mapped[bool] = mapped_column(Boolean, nullable=False, default=True)
    created_at: Mapped[dt.datetime] = mapped_column(DateTime(timezone=True), default=utcnow)
    updated_at: Mapped[dt.datetime] = mapped_column(
        DateTime(timezone=True), default=utcnow, onupdate=utcnow
    )

    last_seen_at: Mapped[dt.datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)
    last_status: Mapped[str] = mapped_column(String(16), nullable=False, default="unknown")
    last_latency_ms: Mapped[float | None] = mapped_column(Float, nullable=True)
    last_error: Mapped[str | None] = mapped_column(Text, nullable=True)

    def __repr__(self) -> str:  # pragma: no cover - 调试用，绝不包含 key
        return f"<Node id={self.id} name={self.name!r} {self.protocol}://{self.address}:{self.port}>"
