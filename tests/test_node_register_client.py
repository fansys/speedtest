from __future__ import annotations

import httpx
import pytest

from app.node_register_client import (
    NodeRegistrationError,
    register_once,
    register_with_retries,
)


def _client(handler) -> httpx.Client:
    return httpx.Client(transport=httpx.MockTransport(handler))


def test_register_once_rejects_non_http_scheme():
    with pytest.raises(NodeRegistrationError):
        register_once(
            "ftp://example.com/register",
            "token",
            name="n",
            address="a",
            port=1,
            protocol="http",
            metadata=None,
            existing_node_key=None,
        )


def test_register_once_succeeds_and_sends_token_header_not_query():
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["url"] = str(request.url)
        seen["token"] = request.headers.get("x-registration-token")
        seen["body"] = request.read()
        return httpx.Response(200, json={"id": 1, "node_key": "abc", "reused": False})

    result = register_once(
        "http://center.local/api/register/auto",
        "reg-token",
        name="n",
        address="a",
        port=1,
        protocol="http",
        metadata={"region": "cn"},
        existing_node_key="old-key",
        client=_client(handler),
    )
    assert result == {"id": 1, "node_key": "abc", "reused": False}
    assert seen["token"] == "reg-token"
    assert "old-key" not in seen["url"]
    assert "reg-token" not in seen["url"]
    assert b"old-key" in seen["body"]


def test_register_once_raises_on_non_200():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(401, json={"detail": "bad token"})

    with pytest.raises(NodeRegistrationError):
        register_once(
            "http://center.local/api/register/auto",
            "wrong",
            name="n",
            address="a",
            port=1,
            protocol="http",
            metadata=None,
            existing_node_key=None,
            client=_client(handler),
        )


def test_register_with_retries_succeeds_after_transient_failures():
    calls = {"n": 0}
    sleeps: list[float] = []

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        if calls["n"] < 3:
            raise httpx.ConnectError("boom", request=request)
        return httpx.Response(200, json={"id": 1, "node_key": "abc", "reused": False})

    result = register_with_retries(
        "http://center.local/api/register/auto",
        "token",
        name="n",
        address="a",
        port=1,
        protocol="http",
        metadata=None,
        existing_node_key=None,
        retries=5,
        client=_client(handler),
        sleep=sleeps.append,
    )
    assert result["node_key"] == "abc"
    assert calls["n"] == 3
    assert sleeps == [1, 2]


def test_register_with_retries_uses_exponential_backoff_capped_at_max_delay():
    sleeps: list[float] = []

    def handler(request: httpx.Request) -> httpx.Response:
        raise httpx.ConnectError("always down", request=request)

    with pytest.raises(NodeRegistrationError):
        register_with_retries(
            "http://center.local/api/register/auto",
            "token",
            name="n",
            address="a",
            port=1,
            protocol="http",
            metadata=None,
            existing_node_key=None,
            retries=6,
            max_delay=16.0,
            client=_client(handler),
            sleep=sleeps.append,
        )
    assert sleeps == [1, 2, 4, 8, 16]


def test_register_with_retries_gives_up_after_max_attempts():
    calls = {"n": 0}

    def handler(request: httpx.Request) -> httpx.Response:
        calls["n"] += 1
        raise httpx.ConnectError("down", request=request)

    with pytest.raises(NodeRegistrationError):
        register_with_retries(
            "http://center.local/api/register/auto",
            "token",
            name="n",
            address="a",
            port=1,
            protocol="http",
            metadata=None,
            existing_node_key=None,
            retries=3,
            client=_client(handler),
            sleep=lambda s: None,
        )
    assert calls["n"] == 3
