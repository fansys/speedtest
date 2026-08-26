from __future__ import annotations

from app.models import Node
from app.security import hash_node_key
from tests.conftest import ADMIN_TOKEN, REGISTRATION_TOKEN


def _register(client, **overrides):
    payload = dict(
        name="节点A",
        address="127.0.0.1",
        port=8081,
        protocol="http",
    )
    payload.update(overrides)
    resp = client.post(
        "/api/register", json=payload, headers={"X-Registration-Token": REGISTRATION_TOKEN}
    )
    assert resp.status_code == 200, resp.text
    body = resp.json()
    return body["node_key"], body


def test_list_requires_admin_token(client):
    assert client.get("/api/nodes").status_code == 401
    assert client.get("/api/nodes", headers={"X-Admin-Token": "wrong"}).status_code == 401


def test_list_never_leaks_node_key(client):
    node_key, node = _register(client)
    resp = client.get("/api/nodes", headers={"X-Admin-Token": ADMIN_TOKEN})
    assert resp.status_code == 200
    body_text = resp.text
    assert node_key not in body_text
    nodes = resp.json()["nodes"]
    assert len(nodes) == 1
    assert nodes[0]["id"] == node["id"]
    assert set(nodes[0].keys()) & {"node_key", "node_key_hash", "node_key_sealed"} == set()


def test_get_node_never_leaks_node_key(client):
    node_key, node = _register(client)
    resp = client.get(f"/api/nodes/{node['id']}", headers={"X-Admin-Token": ADMIN_TOKEN})
    assert resp.status_code == 200
    assert node_key not in resp.text
    assert set(resp.json().keys()) & {"node_key", "node_key_hash", "node_key_sealed"} == set()


def test_database_only_stores_hash_of_node_key(client, app):
    node_key, node = _register(client)
    session_factory = app.state.SessionLocal
    with session_factory() as db:
        db_node = db.get(Node, node["id"])
        assert db_node.node_key_hash == hash_node_key(node_key)
        assert node_key not in (db_node.node_key_hash or "")
        # sealed 副本存在（默认 STORE_NODE_KEY_SEALED=true），但不是明文
        assert db_node.node_key_sealed is not None
        assert node_key not in db_node.node_key_sealed


def test_enable_disable_delete(client):
    _, node = _register(client)
    node_id = node["id"]
    headers = {"X-Admin-Token": ADMIN_TOKEN}

    resp = client.post(f"/api/nodes/{node_id}/disable", headers=headers)
    assert resp.status_code == 200
    assert resp.json()["enabled"] is False

    resp = client.post(f"/api/nodes/{node_id}/enable", headers=headers)
    assert resp.status_code == 200
    assert resp.json()["enabled"] is True

    resp = client.delete(f"/api/nodes/{node_id}", headers=headers)
    assert resp.status_code == 204

    resp = client.get(f"/api/nodes/{node_id}", headers=headers)
    assert resp.status_code == 404


def test_reregistering_same_address_port_invalidates_old_key(client):
    old_key, node = _register(client, port=8090)
    new_key, node2 = _register(client, port=8090, name="节点A-v2")
    assert node2["id"] == node["id"]
    assert new_key != old_key

    headers = {"X-Admin-Token": ADMIN_TOKEN}

    resp = client.post(
        f"/api/nodes/{node['id']}/health", headers=headers, json={"node_key": old_key}
    )
    assert resp.status_code == 400
    assert "不匹配" in resp.json()["detail"]

    # 新 key 通过一致性校验（该端口没有真实监听，连接会失败，但不是 400 不匹配）
    resp = client.post(
        f"/api/nodes/{node['id']}/health", headers=headers, json={"node_key": new_key}
    )
    assert resp.status_code == 200
    assert resp.json()["status"] == "error"
