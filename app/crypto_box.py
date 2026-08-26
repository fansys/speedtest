"""用主密钥（SECRET_KEY）封存 node_key 的小工具。

背景
----
需求要求 node_key「只存哈希」，但同时要求中心服务能主动用节点 key 去访问节点
（健康检查 / 发起测速）。哈希不可逆，因此这两点无法只靠哈希同时满足。

本模块提供第二条轨道：
- 校验永远只依赖 sha256 哈希（见 :mod:`app.security`）；
- 若 ``STORE_NODE_KEY_SEALED=true``（默认），额外保存一份用 SECRET_KEY 加密封存
  的密文，使中心可以自动巡检。密文离开 SECRET_KEY 不可解，数据库泄露本身不会
  泄露 node_key。
- 若设为 false，则数据库里只有哈希，任何对节点的访问都必须由调用方在请求头里
  提供明文 node_key，中心用哈希校验后再转发。

算法：HKDF-SHA256 派生 → HMAC-SHA256 计数器模式产生密钥流做异或 → Encrypt-then-MAC。
全部基于标准库 hmac/hashlib，不引入额外依赖。
"""

from __future__ import annotations

import base64
import hashlib
import hmac
import secrets

_VERSION = b"v1"
_NONCE_LEN = 16
_TAG_LEN = 32
_BLOCK = hashlib.sha256().digest_size


class SealedDataError(ValueError):
    """密文损坏、被篡改，或 SECRET_KEY 与封存时不一致。"""


def _hkdf(secret: bytes, salt: bytes, info: bytes, length: int) -> bytes:
    prk = hmac.new(salt, secret, hashlib.sha256).digest()
    okm = bytearray()
    block = b""
    counter = 1
    while len(okm) < length:
        block = hmac.new(prk, block + info + bytes([counter]), hashlib.sha256).digest()
        okm += block
        counter += 1
    return bytes(okm[:length])


def _derive(secret_key: str, nonce: bytes) -> tuple[bytes, bytes]:
    material = _hkdf(secret_key.encode("utf-8"), nonce, b"librespeed-node-key", 2 * _BLOCK)
    return material[:_BLOCK], material[_BLOCK:]


def _keystream(enc_key: bytes, nonce: bytes, length: int) -> bytes:
    out = bytearray()
    counter = 0
    while len(out) < length:
        out += hmac.new(enc_key, nonce + counter.to_bytes(8, "big"), hashlib.sha256).digest()
        counter += 1
    return bytes(out[:length])


def _xor(data: bytes, pad: bytes) -> bytes:
    return bytes(a ^ b for a, b in zip(data, pad))


def seal(secret_key: str, plaintext: str) -> str:
    """加密封存，返回可直接入库的 base64 字符串。"""
    if not secret_key:
        raise SealedDataError("SECRET_KEY 为空，无法封存 node_key")
    nonce = secrets.token_bytes(_NONCE_LEN)
    enc_key, mac_key = _derive(secret_key, nonce)
    raw = plaintext.encode("utf-8")
    ciphertext = _xor(raw, _keystream(enc_key, nonce, len(raw)))
    tag = hmac.new(mac_key, _VERSION + nonce + ciphertext, hashlib.sha256).digest()
    return base64.urlsafe_b64encode(_VERSION + nonce + tag + ciphertext).decode("ascii")


def unseal(secret_key: str, sealed: str) -> str:
    """解封；密文被篡改或密钥不对时抛出 :class:`SealedDataError`。"""
    if not secret_key:
        raise SealedDataError("SECRET_KEY 为空，无法解封 node_key")
    try:
        blob = base64.urlsafe_b64decode(sealed.encode("ascii"))
    except Exception as exc:  # noqa: BLE001 - 统一成一种错误类型
        raise SealedDataError("封存数据不是合法的 base64") from exc

    header_len = len(_VERSION) + _NONCE_LEN + _TAG_LEN
    if len(blob) < header_len or blob[: len(_VERSION)] != _VERSION:
        raise SealedDataError("封存数据格式不正确")

    nonce = blob[len(_VERSION) : len(_VERSION) + _NONCE_LEN]
    tag = blob[len(_VERSION) + _NONCE_LEN : header_len]
    ciphertext = blob[header_len:]

    enc_key, mac_key = _derive(secret_key, nonce)
    expected = hmac.new(mac_key, _VERSION + nonce + ciphertext, hashlib.sha256).digest()
    if not hmac.compare_digest(tag, expected):
        raise SealedDataError("封存数据校验失败（SECRET_KEY 不匹配或数据被篡改）")

    return _xor(ciphertext, _keystream(enc_key, nonce, len(ciphertext))).decode("utf-8")
