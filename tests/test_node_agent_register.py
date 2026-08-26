from __future__ import annotations

import pytest

from app.node_agent import _build_parser, resolve_node_key
from app.node_state import load_node_state
from tests.conftest import REGISTRATION_TOKEN, make_settings


def _parse(args: list[str]):
    return _build_parser().parse_args(args)


def test_explicit_node_key_skips_registration(tmp_path):
    args = _parse(["--node-key", "explicit-key-value-1234567890"])
    assert resolve_node_key(args) == "explicit-key-value-1234567890"


def test_no_register_url_and_no_key_exits():
    args = _parse([])
    with pytest.raises(SystemExit):
        resolve_node_key(args)


def test_register_url_without_token_exits():
    args = _parse(["--register-url", "http://127.0.0.1:1/api/register/auto", "--address", "127.0.0.1"])
    with pytest.raises(SystemExit):
        resolve_node_key(args)


def test_register_url_without_address_exits():
    args = _parse(["--register-url", "http://127.0.0.1:1/api/register/auto", "--registration-token", "t"])
    with pytest.raises(SystemExit):
        resolve_node_key(args)


def test_invalid_metadata_json_exits():
    args = _parse(
        [
            "--register-url", "http://127.0.0.1:1/api/register/auto",
            "--registration-token", "t",
            "--address", "127.0.0.1",
            "--metadata-json", "{not-json",
        ]
    )
    with pytest.raises(SystemExit):
        resolve_node_key(args)


def test_auto_register_against_live_web_service_writes_node_ini(tmp_path, live_web_service):
    settings = make_settings(tmp_path)
    svc = live_web_service(settings)
    node_ini = tmp_path / "node.ini"

    args = _parse(
        [
            "--register-url", f"{svc.base_url}/api/register/auto",
            "--registration-token", REGISTRATION_TOKEN,
            "--name", "agent-test",
            "--address", "127.0.0.1",
            "--port", "9101",
            "--node-ini", str(node_ini),
            "--register-retries", "3",
        ]
    )
    key = resolve_node_key(args)
    assert key
    state = load_node_state(node_ini)
    assert state["node_key"] == key
    assert state["name"] == "agent-test"
    assert state["address"] == "127.0.0.1"
    assert node_ini.stat().st_mode & 0o777 == 0o600


def test_auto_register_reuses_key_on_second_start(tmp_path, live_web_service):
    settings = make_settings(tmp_path)
    svc = live_web_service(settings)
    node_ini = tmp_path / "node.ini"

    def build_args():
        return _parse(
            [
                "--register-url", f"{svc.base_url}/api/register/auto",
                "--registration-token", REGISTRATION_TOKEN,
                "--name", "agent-test",
                "--address", "127.0.0.1",
                "--port", "9102",
                "--node-ini", str(node_ini),
                "--register-retries", "3",
            ]
        )

    first_key = resolve_node_key(build_args())
    second_key = resolve_node_key(build_args())
    assert first_key == second_key


def test_auto_register_fails_fast_when_server_unreachable(tmp_path):
    args = _parse(
        [
            "--register-url", "http://127.0.0.1:1/api/register/auto",
            "--registration-token", "some-token",
            "--name", "agent-test",
            "--address", "127.0.0.1",
            "--port", "9103",
            "--node-ini", str(tmp_path / "node.ini"),
            "--register-retries", "1",
        ]
    )
    with pytest.raises(SystemExit):
        resolve_node_key(args)
