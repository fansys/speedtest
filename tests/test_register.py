from __future__ import annotations

from app.models import Node
from app.security import generate_node_key, hash_node_key
from tests.conftest import REGISTRATION_TOKEN


def _payload(**overrides):
    payload = dict(
        name="测试节点",
        address="127.0.0.1",
        port=8081,
        protocol="http",
        metadata={"region": "cn-east"},
    )
    payload.update(overrides)
    return payload


def test_register_without_token_rejected(client):
    resp = client.post("/api/register", json=_payload())
    assert resp.status_code == 401


def test_register_with_wrong_token_rejected(client):
    resp = client.post(
        "/api/register",
        json=_payload(),
        headers={"X-Registration-Token": "totally-wrong-token-value"},
    )
    assert resp.status_code == 401


def test_register_without_node_key_succeeds_and_returns_one_time_key(client):
    resp = client.post(
        "/api/register",
        json=_payload(),
        headers={"X-Registration-Token": REGISTRATION_TOKEN},
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["name"] == "测试节点"
    assert body["address"] == "127.0.0.1"
    assert body["port"] == 8081
    assert body["metadata"] == {"region": "cn-east"}

    node_key = body["node_key"]
    assert len(node_key) >= 32
    assert "node_key_hash" not in body
    assert "node_key_sealed" not in body
    assert body["key_fingerprint"] == hash_node_key(node_key)[:12]


def test_register_supports_bearer_header(client):
    resp = client.post(
        "/api/register",
        json=_payload(),
        headers={"Authorization": f"Bearer {REGISTRATION_TOKEN}"},
    )
    assert resp.status_code == 200


def test_register_rejects_disallowed_protocol(client):
    resp = client.post(
        "/api/register",
        json=_payload(protocol="ftp"),
        headers={"X-Registration-Token": REGISTRATION_TOKEN},
    )
    assert resp.status_code == 400


def test_register_ignores_client_supplied_node_key(client):
    """即便客户端在请求体里塞了 node_key，也必须被忽略，服务端仍自动生成。"""
    client_supplied = generate_node_key()
    resp = client.post(
        "/api/register",
        json=_payload(node_key=client_supplied),
        headers={"X-Registration-Token": REGISTRATION_TOKEN},
    )
    assert resp.status_code == 200, resp.text
    assert resp.json()["node_key"] != client_supplied


def test_register_same_address_port_replaces_key(client, app):
    headers = {"X-Registration-Token": REGISTRATION_TOKEN}
    first = client.post("/api/register", json=_payload(name="第一次"), headers=headers)
    assert first.status_code == 200, first.text
    first_body = first.json()
    old_key = first_body["node_key"]
    node_id = first_body["id"]

    second = client.post(
        "/api/register", json=_payload(name="第二次", port=8081), headers=headers
    )
    assert second.status_code == 200, second.text
    second_body = second.json()
    assert second_body["id"] == node_id
    assert second_body["name"] == "第二次"

    new_key = second_body["node_key"]
    assert new_key != old_key

    session_factory = app.state.SessionLocal
    with session_factory() as db:
        db_node = db.get(Node, node_id)
        assert db_node.node_key_hash == hash_node_key(new_key)
        assert db_node.node_key_hash != hash_node_key(old_key)
