from __future__ import annotations

from app.crypto_box import SealedDataError, seal, unseal
from app.security import (
    constant_time_equals,
    extract_bearer,
    generate_node_key,
    hash_node_key,
    key_fingerprint,
)


def test_generate_node_key_is_high_entropy_and_unique():
    a = generate_node_key()
    b = generate_node_key()
    assert a != b
    assert len(a) >= 32


def test_hash_is_deterministic_and_not_reversible_lookalike():
    key = generate_node_key()
    h1 = hash_node_key(key)
    h2 = hash_node_key(key)
    assert h1 == h2
    assert h1 != key
    assert len(h1) == 64  # sha256 hex


def test_fingerprint_is_prefix_of_hash():
    key = generate_node_key()
    assert key_fingerprint(key) == hash_node_key(key)[:12]


def test_constant_time_equals():
    assert constant_time_equals("abc", "abc")
    assert not constant_time_equals("abc", "abd")


def test_extract_bearer():
    assert extract_bearer("Bearer secret-value") == "secret-value"
    assert extract_bearer("bearer secret-value") == "secret-value"
    assert extract_bearer("Basic xxx") is None
    assert extract_bearer(None) is None
    assert extract_bearer("Bearer ") is None


def test_seal_roundtrip():
    secret = "a-secret-key-that-is-long-enough"
    plaintext = generate_node_key()
    sealed = seal(secret, plaintext)
    assert plaintext not in sealed
    assert unseal(secret, sealed) == plaintext


def test_unseal_fails_with_wrong_secret():
    plaintext = generate_node_key()
    sealed = seal("secret-key-one-long-enough-abc", plaintext)
    try:
        unseal("secret-key-two-long-enough-xyz", sealed)
        assert False, "应当抛出 SealedDataError"
    except SealedDataError:
        pass


def test_unseal_fails_with_tampered_ciphertext():
    secret = "a-secret-key-that-is-long-enough"
    sealed = seal(secret, generate_node_key())
    tampered = sealed[:-4] + ("A" if sealed[-4] != "A" else "B") + sealed[-3:]
    try:
        unseal(secret, tampered)
        assert False, "应当检测出篡改"
    except SealedDataError:
        pass
