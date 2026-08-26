"""节点 agent 启动时向中心服务发起自助注册的 HTTP 客户端。

只允许 http/https 注册地址；令牌只走请求头；node_key/existing_node_key 只放在
JSON 请求体里，绝不出现在 query string 或日志中。
"""

from __future__ import annotations

import time
from typing import Any, Callable
from urllib.parse import urlparse

import httpx

DEFAULT_TIMEOUT = 10.0
DEFAULT_MAX_DELAY = 16.0


class NodeRegistrationError(RuntimeError):
    """自动注册失败：网络错误、令牌错误、地址不被允许、响应格式不对等。"""


def _validate_register_url(url: str) -> None:
    parsed = urlparse(url)
    if parsed.scheme not in ("http", "https"):
        raise NodeRegistrationError(f"NODE_REGISTER_URL 只允许 http/https，收到协议: {parsed.scheme or '(空)'}")
    if not parsed.netloc:
        raise NodeRegistrationError("NODE_REGISTER_URL 不是合法的 URL")


def register_once(
    register_url: str,
    registration_token: str,
    *,
    name: str,
    address: str,
    port: int,
    protocol: str,
    metadata: dict | None,
    existing_node_key: str | None,
    timeout: float = DEFAULT_TIMEOUT,
    client: httpx.Client | None = None,
) -> dict[str, Any]:
    """发起一次注册请求，失败抛出 :class:`NodeRegistrationError`。"""
    _validate_register_url(register_url)

    payload: dict[str, Any] = {"name": name, "address": address, "port": port, "protocol": protocol}
    if metadata:
        payload["metadata"] = metadata
    if existing_node_key:
        payload["existing_node_key"] = existing_node_key

    headers = {"X-Registration-Token": registration_token}
    try:
        if client is not None:
            resp = client.post(register_url, json=payload, headers=headers, timeout=timeout)
        else:
            resp = httpx.post(register_url, json=payload, headers=headers, timeout=timeout)
    except httpx.HTTPError as exc:
        raise NodeRegistrationError(f"注册请求失败: {exc}") from exc

    if resp.status_code != 200:
        raise NodeRegistrationError(f"注册被服务端拒绝: HTTP {resp.status_code}")

    try:
        return resp.json()
    except ValueError as exc:
        raise NodeRegistrationError("注册响应不是合法 JSON") from exc


def register_with_retries(
    register_url: str,
    registration_token: str,
    *,
    name: str,
    address: str,
    port: int,
    protocol: str,
    metadata: dict | None,
    existing_node_key: str | None,
    retries: int = 10,
    timeout: float = DEFAULT_TIMEOUT,
    max_delay: float = DEFAULT_MAX_DELAY,
    sleep: Callable[[float], None] = time.sleep,
    client: httpx.Client | None = None,
) -> dict[str, Any]:
    """指数退避重试注册：1、2、4、8、16... 秒（封顶 ``max_delay``）。"""
    retries = max(1, retries)
    last_exc: NodeRegistrationError | None = None
    for attempt in range(1, retries + 1):
        try:
            return register_once(
                register_url,
                registration_token,
                name=name,
                address=address,
                port=port,
                protocol=protocol,
                metadata=metadata,
                existing_node_key=existing_node_key,
                timeout=timeout,
                client=client,
            )
        except NodeRegistrationError as exc:
            last_exc = exc
            if attempt < retries:
                sleep(min(2 ** (attempt - 1), max_delay))
    assert last_exc is not None
    raise last_exc
