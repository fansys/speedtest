from __future__ import annotations

import stat

from app.node_state import load_node_state, save_node_state


def test_load_returns_none_when_file_missing(tmp_path):
    assert load_node_state(tmp_path / "node.ini") is None


def test_save_and_load_round_trip(tmp_path):
    path = tmp_path / "node.ini"
    save_node_state(
        path,
        node_id=1,
        node_key="abc123",
        name="节点A",
        address="127.0.0.1",
        port=8081,
        protocol="http",
        updated_at="2026-08-16T00:00:00+00:00",
    )
    state = load_node_state(path)
    assert state == {
        "node_id": "1",
        "node_key": "abc123",
        "name": "节点A",
        "address": "127.0.0.1",
        "port": "8081",
        "protocol": "http",
        "updated_at": "2026-08-16T00:00:00+00:00",
    }


def test_save_creates_parent_directory(tmp_path):
    path = tmp_path / "nested" / "dir" / "node.ini"
    save_node_state(path, node_id=1, node_key="k")
    assert path.is_file()


def test_save_sets_permissions_to_0600(tmp_path):
    path = tmp_path / "node.ini"
    save_node_state(path, node_id=1, node_key="k")
    mode = stat.S_IMODE(path.stat().st_mode)
    assert mode == 0o600


def test_save_overwrites_previous_content(tmp_path):
    path = tmp_path / "node.ini"
    save_node_state(path, node_id=1, node_key="old-key")
    save_node_state(path, node_id=1, node_key="new-key")
    state = load_node_state(path)
    assert state["node_key"] == "new-key"


def test_save_skips_none_values(tmp_path):
    path = tmp_path / "node.ini"
    save_node_state(path, node_id=1, node_key="k", name=None)
    state = load_node_state(path)
    assert "name" not in state


def test_save_leaves_no_temp_files_behind(tmp_path):
    path = tmp_path / "node.ini"
    save_node_state(path, node_id=1, node_key="k")
    leftovers = [p for p in tmp_path.iterdir() if p.name != "node.ini"]
    assert leftovers == []
