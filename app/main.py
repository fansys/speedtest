"""中心 Web 服务的 FastAPI 应用工厂。

启动方式（生产 / 开发）:

    uvicorn app.main:create_app --factory --host 0.0.0.0 --port 8080

使用工厂函数而不是模块级单例，是为了让 :func:`get_settings` 的校验（缺失
ADMIN_TOKEN 等会直接抛异常）只在真正启动/构造应用时发生，测试可以传入
独立的 ``Settings`` 而不受影响。
"""

from __future__ import annotations

from pathlib import Path

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from fastapi.responses import FileResponse

from .config import Settings, get_settings
from .db import create_engine_and_sessionmaker, init_db
from .routers import nodes, register

STATIC_DIR = Path(__file__).resolve().parent.parent / "static"


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or get_settings()

    engine, session_factory = create_engine_and_sessionmaker(settings.database_url)
    init_db(engine)

    app = FastAPI(title="LibreSpeed 测速服务", version="0.1.0")
    app.state.settings = settings
    app.state.engine = engine
    app.state.SessionLocal = session_factory

    app.include_router(register.router)
    app.include_router(nodes.router)

    if STATIC_DIR.is_dir():
        app.mount("/static", StaticFiles(directory=str(STATIC_DIR)), name="static")

        @app.get("/", include_in_schema=False)
        def index() -> FileResponse:
            return FileResponse(str(STATIC_DIR / "index.html"))

    @app.get("/healthz", include_in_schema=False)
    def service_health() -> dict:
        return {"status": "ok"}

    return app
