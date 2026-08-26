from __future__ import annotations

from fastapi.testclient import TestClient

from app.main import create_app
from app.security import generate_node_key
from tests.conftest import ADMIN_TOKEN, REGISTRATION_TOKEN, make_settings


def _register(client, address, port, **overrides):
    payload = dict(name="真实节点", address=address, port=port, protocol="http")
    payload.update(overrides)
    resp = client.post(
        "/api/register", json=payload, headers={"X-Registration-Token": REGISTRATION_TOKEN}
    )
    assert resp.status_code == 200, resp.text
    return resp.json()


def test_health_check_against_live_node(tmp_path, live_node_agent, free_port):
    settings = make_settings(tmp_path)
    app = create_app(settings)
    client = TestClient(app)

    port = free_port()
    node = _register(client, "127.0.0.1", port)
    live_node_agent(node["node_key"], port=port)

    resp = client.post(
        f"/api/nodes/{node['id']}/health", headers={"X-Admin-Token": ADMIN_TOKEN}
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["status"] == "online"
    assert body["latency_ms"] is not None


def test_speedtest_against_live_node_reports_ping_download_upload(
    tmp_path, live_node_agent, free_port
):
    settings = make_settings(tmp_path)
    app = create_app(settings)
    client = TestClient(app)

    port = free_port()
    node = _register(client, "127.0.0.1", port)
    live_node_agent(node["node_key"], port=port)

    resp = client.post(
        f"/api/nodes/{node['id']}/speedtest",
        headers={"X-Admin-Token": ADMIN_TOKEN},
        json={"ping_count": 2, "download_bytes": 200_000, "upload_bytes": 100_000},
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["error"] is None
    assert body["ping"]["count"] == 2
    assert body["download"]["bytes"] == 200_000
    assert body["download"]["mbps"] >= 0
    assert body["upload"]["bytes"] == 100_000
    assert body["upload"]["mbps"] >= 0


def test_speedtest_without_sealed_key_requires_explicit_node_key(
    tmp_path, live_node_agent, free_port
):
    settings = make_settings(tmp_path, store_node_key_sealed=False)
    app = create_app(settings)
    client = TestClient(app)

    port = free_port()
    node = _register(client, "127.0.0.1", port)
    node_key = node["node_key"]
    live_node_agent(node_key, port=port)

    # 未提供 node_key：应当明确报错，而不是静默失败
    resp = client.post(
        f"/api/nodes/{node['id']}/speedtest", headers={"X-Admin-Token": ADMIN_TOKEN}, json={}
    )
    assert resp.status_code == 400
    assert "node_key" in resp.json()["detail"]

    # 提供正确 node_key：应当成功
    resp = client.post(
        f"/api/nodes/{node['id']}/speedtest",
        headers={"X-Admin-Token": ADMIN_TOKEN},
        json={"node_key": node_key, "download_bytes": 50_000, "upload_bytes": 50_000, "ping_count": 1},
    )
    assert resp.status_code == 200, resp.text
    assert resp.json()["error"] is None


def test_speedtest_with_mismatched_node_key_rejected(tmp_path, live_node_agent, free_port):
    settings = make_settings(tmp_path, store_node_key_sealed=False)
    app = create_app(settings)
    client = TestClient(app)

    port = free_port()
    node = _register(client, "127.0.0.1", port)
    live_node_agent(node["node_key"], port=port)

    resp = client.post(
        f"/api/nodes/{node['id']}/speedtest",
        headers={"X-Admin-Token": ADMIN_TOKEN},
        json={"node_key": generate_node_key()},
    )
    assert resp.status_code == 400
    assert "不匹配" in resp.json()["detail"]


def test_speedtest_on_disabled_node_rejected(tmp_path, live_node_agent, free_port):
    settings = make_settings(tmp_path)
    app = create_app(settings)
    client = TestClient(app)

    port = free_port()
    node = _register(client, "127.0.0.1", port)
    live_node_agent(node["node_key"], port=port)
    client.post(f"/api/nodes/{node['id']}/disable", headers={"X-Admin-Token": ADMIN_TOKEN})

    resp = client.post(
        f"/api/nodes/{node['id']}/speedtest", headers={"X-Admin-Token": ADMIN_TOKEN}
    )
    assert resp.status_code == 409


def test_speedtest_reports_error_when_node_unreachable(tmp_path):
    settings = make_settings(tmp_path)
    app = create_app(settings)
    client = TestClient(app)

    # 注册一个没有实际监听的端口
    node = _register(client, "127.0.0.1", 8, protocol="http")

    resp = client.post(
        f"/api/nodes/{node['id']}/speedtest", headers={"X-Admin-Token": ADMIN_TOKEN}
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["error"] is not None
