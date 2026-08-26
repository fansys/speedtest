from __future__ import annotations

from fastapi.testclient import TestClient

from app.node_agent import create_node_app
from app.security import generate_node_key


def test_node_agent_rejects_missing_key():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    assert client.get("/healthz").status_code == 401
    assert client.get("/ping").status_code == 401
    assert client.get("/download?bytes=100").status_code == 401
    assert client.post("/upload", content=b"abc").status_code == 401


def test_node_agent_rejects_wrong_key():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    resp = client.get("/healthz", headers={"X-Node-Key": "wrong-key-value"})
    assert resp.status_code == 401
    assert node_key not in resp.text


def test_node_agent_accepts_correct_key_header():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    resp = client.get("/healthz", headers={"X-Node-Key": node_key})
    assert resp.status_code == 200
    assert resp.json()["status"] == "ok"


def test_node_agent_accepts_bearer_header():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    resp = client.get("/healthz", headers={"Authorization": f"Bearer {node_key}"})
    assert resp.status_code == 200


def test_node_agent_ping_is_fast_and_empty():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    resp = client.get("/ping", headers={"X-Node-Key": node_key})
    assert resp.status_code == 204


def test_node_agent_download_returns_requested_bytes():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    resp = client.get("/download?bytes=12345", headers={"X-Node-Key": node_key})
    assert resp.status_code == 200
    assert len(resp.content) == 12345


def test_node_agent_download_respects_max_test_bytes():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1", max_test_bytes=1000)
    client = TestClient(app)
    resp = client.get("/download?bytes=999999", headers={"X-Node-Key": node_key})
    assert resp.status_code == 200
    assert len(resp.content) == 1000


def test_node_agent_upload_reports_bytes():
    node_key = generate_node_key()
    app = create_node_app(node_key, "n1")
    client = TestClient(app)
    payload = b"x" * 5000
    resp = client.post("/upload", content=payload, headers={"X-Node-Key": node_key})
    assert resp.status_code == 200
    body = resp.json()
    assert body["bytes"] == 5000
    assert body["duration_ms"] >= 0
    assert body["mbps"] >= 0
