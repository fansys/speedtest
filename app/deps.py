"""跨路由复用的 FastAPI 依赖。"""

from __future__ import annotations

from typing import Iterator

from fastapi import Request
from sqlalchemy.orm import Session

from .config import Settings


def get_db(request: Request) -> Iterator[Session]:
    session_factory = request.app.state.SessionLocal
    db = session_factory()
    try:
        yield db
    finally:
        db.close()


def get_settings_dep(request: Request) -> Settings:
    return request.app.state.settings
