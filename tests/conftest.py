from __future__ import annotations

import socket
import threading
import time

import pytest
import uvicorn
from fastapi.testclient import TestClient

from app.config import Settings
from app.main import create_app
from app.node_agent import create_node_app

ADMIN_TOKEN = "admin-token-1234567890"
REGISTRATION_TOKEN = "registration-token-1234567890"
SECRET_KEY = "secret-key-1234567890abcdef"


def make_settings(tmp_path, **overrides) -> Settings:
    db_path = tmp_path / "test.db"
    kwargs = dict(
        admin_token=ADMIN_TOKEN,
        registration_token=REGISTRATION_TOKEN,
        secret_key=SECRET_KEY,
        database_url=f"sqlite:///{db_path}",
        allow_private_nodes=True,
    )
    kwargs.update(overrides)
    return Settings(_env_file=None, **kwargs)


@pytest.fixture
def settings(tmp_path) -> Settings:
    return make_settings(tmp_path)


@pytest.fixture
def app(settings):
    return create_app(settings)


@pytest.fixture
def client(app) -> TestClient:
    return TestClient(app)


def _free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


class LiveNodeAgent:
    """在后台线程里跑一个真实的 uvicorn 服务，供端到端测速测试使用。"""

    def __init__(self, node_key: str, name: str = "test-node", port: int | None = None):
        self.node_key = node_key
        self.port = port if port is not None else _free_port()
        app = create_node_app(node_key, name)
        config = uvicorn.Config(app, host="127.0.0.1", port=self.port, log_level="warning")
        self.server = uvicorn.Server(config)
        self.thread = threading.Thread(target=self.server.run, daemon=True)

    def start(self) -> None:
        self.thread.start()
        for _ in range(200):
            if getattr(self.server, "started", False):
                return
            time.sleep(0.02)
        raise RuntimeError("node agent 未能在预期时间内启动")

    def stop(self) -> None:
        self.server.should_exit = True
        self.thread.join(timeout=5)


@pytest.fixture
def free_port():
    return _free_port


@pytest.fixture
def live_node_agent():
    agents: list[LiveNodeAgent] = []

    def _start(node_key: str, name: str = "test-node", port: int | None = None) -> LiveNodeAgent:
        agent = LiveNodeAgent(node_key, name, port=port)
        agent.start()
        agents.append(agent)
        return agent

    yield _start

    for agent in agents:
        agent.stop()


class LiveWebService:
    """在后台线程里跑一个真实的中心服务 uvicorn 实例，供节点 agent 自动注册端到端测试使用。"""

    def __init__(self, settings: Settings):
        self.settings = settings
        self.port = _free_port()
        app = create_app(settings)
        config = uvicorn.Config(app, host="127.0.0.1", port=self.port, log_level="warning")
        self.server = uvicorn.Server(config)
        self.thread = threading.Thread(target=self.server.run, daemon=True)

    @property
    def base_url(self) -> str:
        return f"http://127.0.0.1:{self.port}"

    def start(self) -> None:
        self.thread.start()
        for _ in range(200):
            if getattr(self.server, "started", False):
                return
            time.sleep(0.02)
        raise RuntimeError("web 服务未能在预期时间内启动")

    def stop(self) -> None:
        self.server.should_exit = True
        self.thread.join(timeout=5)


@pytest.fixture
def live_web_service():
    services: list[LiveWebService] = []

    def _start(settings: Settings) -> LiveWebService:
        svc = LiveWebService(settings)
        svc.start()
        services.append(svc)
        return svc

    yield _start

    for svc in services:
        svc.stop()
