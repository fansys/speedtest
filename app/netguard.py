"""SSRF 防护：校验节点 address/port 是否允许被中心服务访问。

节点地址是「管理员输入」，中心服务会主动向它发起请求，属于典型 SSRF 面。
这里做的事：

1. 只允许 http / https（由 ALLOWED_NODE_PROTOCOLS 控制），拒绝 file/gopher/ftp 等。
2. 拒绝带认证信息、路径、查询串的地址（address 只接受主机名或 IP 字面量）。
3. 端口必须在 1-65535，且不在明显危险的端口黑名单里。
4. 解析主机名得到全部 A/AAAA 记录，逐个检查：回环、私网、链路本地、保留、
   多播、未指定地址默认按 ALLOW_PRIVATE_NODES 开关处理（MVP 默认允许，
   因为内网自建测速节点是主要场景）。
5. 调用方在真正发请求前会再次校验（见 node_client），缩小 DNS rebinding 窗口。
"""

from __future__ import annotations

import ipaddress
import socket
from dataclasses import dataclass, field

# 这些端口即使在 ALLOW_PRIVATE_NODES=true 时也一律拒绝：
# 它们不是 HTTP 服务端口，指向它们通常意味着在拿本服务当跳板打内网组件。
BLOCKED_PORTS: frozenset[int] = frozenset(
    {
        22,  # ssh
        23,  # telnet
        25,  # smtp
        445,  # smb
        587,  # smtp submission
        3306,  # mysql
        5432,  # postgres
        6379,  # redis
        9200,  # elasticsearch
        11211,  # memcached
        27017,  # mongodb
    }
)


class NodeAddressError(ValueError):
    """节点地址不被允许。错误信息面向管理员，不含任何令牌。"""


@dataclass(frozen=True)
class ResolvedNode:
    protocol: str
    host: str
    port: int
    addresses: tuple[str, ...] = field(default=())

    @property
    def base_url(self) -> str:
        host = f"[{self.host}]" if ":" in self.host else self.host
        return f"{self.protocol}://{host}:{self.port}"


def _classify(ip: ipaddress.IPv4Address | ipaddress.IPv6Address) -> str | None:
    """返回不安全的原因；安全则返回 None。"""
    if ip.is_unspecified:
        return "未指定地址（0.0.0.0 / ::）"
    if ip.is_loopback:
        return "回环地址"
    if ip.is_link_local:
        return "链路本地地址（含云元数据 169.254.169.254）"
    if ip.is_multicast:
        return "多播地址"
    if ip.is_reserved:
        return "保留地址"
    if ip.version == 6:
        if ip.is_site_local:
            return "IPv6 site-local 地址"
        mapped = getattr(ip, "ipv4_mapped", None)
        if mapped is not None:
            return _classify(mapped)
    if ip.is_private:
        return "私网地址"
    return None


def resolve_addresses(host: str) -> list[str]:
    """解析主机名到 IP 列表；host 本身是 IP 时直接返回。"""
    try:
        ipaddress.ip_address(host)
        return [host]
    except ValueError:
        pass
    try:
        infos = socket.getaddrinfo(host, None, proto=socket.IPPROTO_TCP)
    except socket.gaierror as exc:
        raise NodeAddressError(f"无法解析主机名 {host!r}") from exc
    seen: list[str] = []
    for info in infos:
        addr = info[4][0]
        if addr not in seen:
            seen.append(addr)
    if not seen:
        raise NodeAddressError(f"主机名 {host!r} 没有解析到任何地址")
    return seen


def validate_node_target(
    address: str,
    port: int,
    protocol: str,
    *,
    allow_private: bool,
    allowed_protocols: list[str],
    resolve: bool = True,
) -> ResolvedNode:
    """校验节点地址，返回归一化后的结果。不通过则抛 :class:`NodeAddressError`。"""
    protocol = (protocol or "").strip().lower()
    if protocol not in allowed_protocols:
        raise NodeAddressError(
            f"protocol 必须是 {'/'.join(allowed_protocols)} 之一，收到 {protocol or '(空)'}"
        )

    host = (address or "").strip()
    if not host:
        raise NodeAddressError("address 不能为空")
    if "://" in host:
        raise NodeAddressError("address 只填主机名或 IP，不要带协议前缀")
    for bad in ("/", "?", "#", "@", "\\", " "):
        if bad in host:
            raise NodeAddressError(f"address 中不允许出现字符 {bad!r}")
    # IPv6 字面量允许用方括号包裹
    if host.startswith("[") and host.endswith("]"):
        host = host[1:-1]
    if ":" in host:
        try:
            ipaddress.IPv6Address(host)
        except ValueError as exc:
            raise NodeAddressError("address 中不允许出现端口号或非法的 IPv6 字面量") from exc
    if len(host) > 253:
        raise NodeAddressError("address 过长")

    if not isinstance(port, int) or isinstance(port, bool):
        raise NodeAddressError("port 必须是整数")
    if not 1 <= port <= 65535:
        raise NodeAddressError("port 必须在 1-65535 之间")
    if port in BLOCKED_PORTS:
        raise NodeAddressError(f"端口 {port} 在禁止列表中（非 HTTP 服务端口）")

    if not resolve:
        return ResolvedNode(protocol=protocol, host=host, port=port)

    addresses = resolve_addresses(host)
    if not allow_private:
        for addr in addresses:
            reason = _classify(ipaddress.ip_address(addr))
            if reason is not None:
                raise NodeAddressError(
                    f"地址 {addr} 属于{reason}，当前 ALLOW_PRIVATE_NODES=false，已拒绝"
                )

    return ResolvedNode(protocol=protocol, host=host, port=port, addresses=tuple(addresses))
