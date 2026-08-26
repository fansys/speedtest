"""令牌生成、哈希与恒定时间校验。

约定：
- 所有令牌只通过请求头传递（X-Admin-Token / X-Registration-Token / X-Node-Key
  或 Authorization: Bearer），绝不接受 query string，避免进入访问日志。
- node_key 只存 sha256 哈希；比较一律使用 hmac.compare_digest。
"""

from __future__ import annotations

import hashlib
import hmac
import secrets

from fastapi import Header, HTTPException, Request, status

NODE_KEY_BYTES = 32
"""node_key 的随机字节数（256 bit 熵）。"""

MIN_NODE_KEY_LENGTH = 24
"""接受外部提交的 node_key 时要求的最小长度。"""


def generate_node_key() -> str:
    """生成高熵 node_key。"""
    return secrets.token_urlsafe(NODE_KEY_BYTES)


def hash_node_key(node_key: str) -> str:
    """node_key 的 sha256 十六进制摘要。

    node_key 本身是 256bit 均匀随机值，不存在字典/暴力风险，
    因此使用快速哈希而非 KDF（避免每次请求付出 KDF 代价）。
    """
    return hashlib.sha256(node_key.encode("utf-8")).hexdigest()


def key_fingerprint(node_key: str) -> str:
    """可安全展示的短指纹，用于在 UI 上区分 key，不可反推原文。"""
    return hash_node_key(node_key)[:12]


def constant_time_equals(a: str, b: str) -> bool:
    return hmac.compare_digest(a.encode("utf-8"), b.encode("utf-8"))


def extract_bearer(authorization: str | None) -> str | None:
    if not authorization:
        return None
    scheme, _, value = authorization.partition(" ")
    if scheme.lower() != "bearer" or not value.strip():
        return None
    return value.strip()


def _pick_token(header_value: str | None, authorization: str | None) -> str | None:
    return header_value.strip() if header_value and header_value.strip() else extract_bearer(authorization)


def _unauthorized(detail: str) -> HTTPException:
    # detail 中永远不回显收到的令牌内容。
    return HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail=detail)


async def require_admin(
    request: Request,
    x_admin_token: str | None = Header(default=None, alias="X-Admin-Token"),
    authorization: str | None = Header(default=None),
) -> None:
    """管理 API 鉴权。"""
    settings = request.app.state.settings
    supplied = _pick_token(x_admin_token, authorization)
    if not supplied or not constant_time_equals(supplied, settings.admin_token):
        raise _unauthorized("管理令牌无效或缺失")


async def require_registration(
    request: Request,
    x_registration_token: str | None = Header(default=None, alias="X-Registration-Token"),
    authorization: str | None = Header(default=None),
) -> None:
    """节点注册鉴权。"""
    settings = request.app.state.settings
    supplied = _pick_token(x_registration_token, authorization)
    if not supplied or not constant_time_equals(supplied, settings.registration_token):
        raise _unauthorized("注册令牌无效或缺失")


def read_node_key_header(
    x_node_key: str | None,
    authorization: str | None,
) -> str | None:
    """从请求头中取出 node_key（node agent 与中心服务共用）。"""
    return _pick_token(x_node_key, authorization)
