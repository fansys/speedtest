"""中心服务作为「测速客户端」去访问节点 agent。

每次真正发起请求前都会重新调用 :func:`app.netguard.validate_node_target`，
缩小 DNS rebinding 的窗口；调用一律携带该节点专属的 node_key。
"""

from __future__ import annotations

import secrets
import time
from typing import Any

import httpx

from .config import Settings
from .models import Node
from .netguard import NodeAddressError, validate_node_target


class NodeClientError(RuntimeError):
    """节点不可达、拒绝了 node_key，或返回了非预期响应。"""


def _base_url(node: Node, settings: Settings) -> str:
    try:
        resolved = validate_node_target(
            node.address,
            node.port,
            node.protocol,
            allow_private=settings.allow_private_nodes,
            allowed_protocols=settings.protocol_list,
            resolve=True,
        )
    except NodeAddressError as exc:
        raise NodeClientError(f"节点地址不再被允许访问: {exc}") from exc
    return resolved.base_url


def _headers(node_key: str) -> dict[str, str]:
    return {"X-Node-Key": node_key}


async def check_health(node: Node, node_key: str, settings: Settings) -> dict[str, Any]:
    base = _base_url(node, settings)
    async with httpx.AsyncClient(timeout=settings.node_health_timeout) as client:
        try:
            resp = await client.get(f"{base}/healthz", headers=_headers(node_key))
        except httpx.HTTPError as exc:
            raise NodeClientError(f"无法连接节点: {exc}") from exc
    if resp.status_code == 401:
        raise NodeClientError("节点拒绝了 node_key")
    if resp.status_code != 200:
        raise NodeClientError(f"节点返回异常状态码 {resp.status_code}")
    return resp.json()


async def measure_ping(
    node: Node, node_key: str, settings: Settings, count: int | None = None
) -> dict[str, Any]:
    base = _base_url(node, settings)
    count = count or settings.ping_count
    latencies: list[float] = []
    async with httpx.AsyncClient(timeout=settings.node_connect_timeout) as client:
        for _ in range(count):
            start = time.perf_counter()
            try:
                resp = await client.get(f"{base}/ping", headers=_headers(node_key))
            except httpx.HTTPError as exc:
                raise NodeClientError(f"ping 失败: {exc}") from exc
            if resp.status_code == 401:
                raise NodeClientError("节点拒绝了 node_key")
            if resp.status_code not in (200, 204):
                raise NodeClientError(f"节点返回异常状态码 {resp.status_code}")
            latencies.append((time.perf_counter() - start) * 1000)
    return {
        "count": len(latencies),
        "min_ms": min(latencies),
        "avg_ms": sum(latencies) / len(latencies),
        "max_ms": max(latencies),
        "jitter_ms": max(latencies) - min(latencies),
    }


async def measure_download(
    node: Node, node_key: str, settings: Settings, num_bytes: int | None = None
) -> dict[str, Any]:
    base = _base_url(node, settings)
    num_bytes = min(num_bytes or settings.default_download_bytes, settings.max_test_bytes)
    total = 0
    start = time.perf_counter()
    async with httpx.AsyncClient(timeout=settings.node_request_timeout) as client:
        try:
            async with client.stream(
                "GET",
                f"{base}/download",
                params={"bytes": num_bytes},
                headers=_headers(node_key),
            ) as resp:
                if resp.status_code == 401:
                    raise NodeClientError("节点拒绝了 node_key")
                if resp.status_code != 200:
                    raise NodeClientError(f"节点返回异常状态码 {resp.status_code}")
                async for chunk in resp.aiter_bytes():
                    total += len(chunk)
        except httpx.HTTPError as exc:
            raise NodeClientError(f"下载测速失败: {exc}") from exc
    elapsed = max(time.perf_counter() - start, 1e-9)
    mbps = (total * 8 / 1_000_000) / elapsed
    return {"bytes": total, "duration_ms": elapsed * 1000, "mbps": mbps}


async def measure_upload(
    node: Node, node_key: str, settings: Settings, num_bytes: int | None = None
) -> dict[str, Any]:
    base = _base_url(node, settings)
    num_bytes = min(num_bytes or settings.default_upload_bytes, settings.max_test_bytes)
    chunk_size = min(settings.stream_chunk_bytes, num_bytes) if num_bytes else 0
    payload_chunk = secrets.token_bytes(chunk_size) if chunk_size else b""

    async def body() -> Any:
        sent = 0
        while sent < num_bytes:
            n = min(len(payload_chunk), num_bytes - sent)
            yield payload_chunk[:n]
            sent += n

    start = time.perf_counter()
    async with httpx.AsyncClient(timeout=settings.node_request_timeout) as client:
        try:
            resp = await client.post(
                f"{base}/upload", headers=_headers(node_key), content=body()
            )
        except httpx.HTTPError as exc:
            raise NodeClientError(f"上传测速失败: {exc}") from exc
    if resp.status_code == 401:
        raise NodeClientError("节点拒绝了 node_key")
    if resp.status_code != 200:
        raise NodeClientError(f"节点返回异常状态码 {resp.status_code}")
    elapsed = max(time.perf_counter() - start, 1e-9)
    mbps = (num_bytes * 8 / 1_000_000) / elapsed
    return {"bytes": num_bytes, "duration_ms": elapsed * 1000, "mbps": mbps}
