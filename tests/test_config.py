from __future__ import annotations

import pytest
from pydantic import ValidationError

from app.config import Settings
from tests.conftest import ADMIN_TOKEN, REGISTRATION_TOKEN, SECRET_KEY


def test_missing_tokens_raise(tmp_path):
    with pytest.raises(ValidationError):
        Settings(_env_file=None, database_url=f"sqlite:///{tmp_path/'x.db'}")


def test_short_token_rejected(tmp_path):
    with pytest.raises(ValidationError):
        Settings(
            _env_file=None,
            admin_token="short",
            registration_token=REGISTRATION_TOKEN,
            secret_key=SECRET_KEY,
            database_url=f"sqlite:///{tmp_path/'x.db'}",
        )


def test_admin_and_registration_token_must_differ(tmp_path):
    with pytest.raises(ValidationError):
        Settings(
            _env_file=None,
            admin_token=ADMIN_TOKEN,
            registration_token=ADMIN_TOKEN,
            secret_key=SECRET_KEY,
            database_url=f"sqlite:///{tmp_path/'x.db'}",
        )


def test_sealed_storage_requires_secret_key(tmp_path):
    with pytest.raises(ValidationError):
        Settings(
            _env_file=None,
            admin_token=ADMIN_TOKEN,
            registration_token=REGISTRATION_TOKEN,
            secret_key="",
            store_node_key_sealed=True,
            database_url=f"sqlite:///{tmp_path/'x.db'}",
        )


def test_valid_settings_ok(tmp_path):
    settings = Settings(
        _env_file=None,
        admin_token=ADMIN_TOKEN,
        registration_token=REGISTRATION_TOKEN,
        secret_key=SECRET_KEY,
        database_url=f"sqlite:///{tmp_path/'x.db'}",
    )
    assert settings.admin_token == ADMIN_TOKEN
    assert settings.protocol_list == ["http", "https"]
