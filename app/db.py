"""数据库引擎 / Session 工厂。

不使用全局单例：每个 :func:`app.main.create_app` 调用都会根据传入的
``Settings`` 创建独立的 engine，方便测试之间互不干扰（各用各的 sqlite 文件）。
"""

from __future__ import annotations

from sqlalchemy import create_engine
from sqlalchemy.orm import Session, sessionmaker

from .models import Base


def create_engine_and_sessionmaker(database_url: str):
    connect_args = {"check_same_thread": False} if database_url.startswith("sqlite") else {}
    engine = create_engine(database_url, connect_args=connect_args)
    session_factory = sessionmaker(bind=engine, autoflush=False, autocommit=False, class_=Session)
    return engine, session_factory


def init_db(engine) -> None:
    Base.metadata.create_all(engine)
