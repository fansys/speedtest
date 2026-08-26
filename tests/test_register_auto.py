from __future__ import annotations

from app.models import Node
from app.security import generate_node_key, hash_node_key
from tests.conftest import REGISTRATION_TOKEN


def _payload(**overrides):
    payload = dict(
        name="自动注册节点",
        address="127.0.0.1",
        port=8091,
        protocol="http",
        metadata={"region": "cn-east"},
    )
    payload.update(overrides)
    return payload


def _auto_register(client, **overrides):
    return client.post(
        "/api/register/auto",
        json=_payload(**overrides),
        headers={"X-Registration-Token": REGISTRATION_TOKEN},
    )


def test_auto_register_without_token_rejected(client):
    resp = client.post("/api/register/auto", json=_payload())
    assert resp.status_code == 401


def test_auto_register_first_time_generates_new_key(client):
    resp = _auto_register(client)
    assert resp.status_code == 200, resp.text
    body = resp.json()
    assert body["reused"] is False
    assert len(body["node_key"]) >= 32
    assert "node_key_hash" not in body
    assert "node_key_sealed" not in body


def test_auto_register_reuses_key_for_same_address_port(client):
    first = _auto_register(client).json()
    node_id = first["id"]
    old_key = first["node_key"]

    second = _auto_register(client, existing_node_key=old_key, name="自动注册节点-v2").json()
    assert second["id"] == node_id
    assert second["reused"] is True
    assert second["node_key"] == old_key
    assert second["name"] == "自动注册节点-v2"


def test_auto_register_rotates_when_existing_key_is_wrong(client, app):
    first = _auto_register(client).json()
    node_id = first["id"]
    old_key = first["node_key"]

    bogus_key = generate_node_key()
    second = _auto_register(client, existing_node_key=bogus_key).json()
    assert second["id"] == node_id
    assert second["reused"] is False
    assert second["node_key"] != old_key
    assert second["node_key"] != bogus_key

    session_factory = app.state.SessionLocal
    with session_factory() as db:
        db_node = db.get(Node, node_id)
        assert db_node.node_key_hash == hash_node_key(second["node_key"])
        assert db_node.node_key_hash != hash_node_key(old_key)


def test_auto_register_rotates_after_node_deleted(client, app):
    first = _auto_register(client).json()
    old_key = first["node_key"]
    node_id = first["id"]

    session_factory = app.state.SessionLocal
    with session_factory() as db:
        db.delete(db.get(Node, node_id))
        db.commit()

    second = _auto_register(client, existing_node_key=old_key).json()
    assert second["id"] != node_id
    assert second["reused"] is False
    assert second["node_key"] != old_key


def test_auto_register_cannot_reuse_key_across_different_address_port(client, app):
    """带着 A 节点的 key 去注册 B 地址/端口时，绝不能复用，也绝不能改动 A 节点。"""
    node_a = _auto_register(client, address="127.0.0.1", port=9001, name="节点A").json()
    key_a = node_a["node_key"]

    node_b = _auto_register(client, address="127.0.0.1", port=9002, name="节点B", existing_node_key=key_a).json()
    assert node_b["id"] != node_a["id"]
    assert node_b["reused"] is False
    assert node_b["node_key"] != key_a

    session_factory = app.state.SessionLocal
    with session_factory() as db:
        db_node_a = db.get(Node, node_a["id"])
        # 节点 A 的 key 必须完全不受影响。
        assert db_node_a.node_key_hash == hash_node_key(key_a)
        assert db_node_a.address == "127.0.0.1"
        assert db_node_a.port == 9001


def test_auto_register_plaintext_key_never_leaked_via_admin_api(client):
    from tests.conftest import ADMIN_TOKEN

    body = _auto_register(client).json()
    node_key = body["node_key"]

    resp = client.get("/api/nodes", headers={"X-Admin-Token": ADMIN_TOKEN})
    assert node_key not in resp.text
    nodes = resp.json()["nodes"]
    assert set(nodes[0].keys()) & {"node_key", "node_key_hash", "node_key_sealed"} == set()


def test_auto_register_rejects_disallowed_protocol(client):
    resp = _auto_register(client, protocol="ftp")
    assert resp.status_code == 400


def test_manual_register_still_ignores_existing_node_key_field(client):
    """/api/register 手动接口即便传了 existing_node_key 这样的多余字段也应被忽略。"""
    resp = client.post(
        "/api/register",
        json=_payload(existing_node_key="whatever"),
        headers={"X-Registration-Token": REGISTRATION_TOKEN},
    )
    assert resp.status_code == 200
    assert "reused" not in resp.json()
