from __future__ import annotations

from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent


def _read(name: str) -> str:
    return (ROOT / name).read_text(encoding="utf-8")


def test_web_dockerfile_exposes_and_healthchecks():
    content = _read("Dockerfile")
    assert "EXPOSE 8080" in content
    assert "HEALTHCHECK" in content
    assert "CMD" in content


def test_node_dockerfile_uses_non_root_user_and_healthcheck():
    content = _read("Dockerfile.node")
    assert "HEALTHCHECK" in content
    assert "EXPOSE 8081" in content
    assert "USER nodeagent" in content
    assert "useradd" in content
    for env_var in ("NODE_KEY", "NODE_NAME", "NODE_PORT"):
        assert env_var in content
    assert "app.node_agent" in content


def test_docker_compose_is_valid_yaml_with_web_and_node_services():
    doc = yaml.safe_load(_read("docker-compose.yml"))
    assert "services" in doc
    services = doc["services"]
    assert "web" in services and "node" in services

    web = services["web"]
    assert web["build"]["dockerfile"] == "Dockerfile"
    assert any("/srv/app/data" in v for v in web.get("volumes", []))
    assert web.get("env_file") == [".env"]

    node = services["node"]
    assert node["build"]["dockerfile"] == "Dockerfile.node"
    node_env = node.get("environment", {})
    assert "NODE_KEY" in node_env
    assert "NODE_NAME" in node_env
    assert "NODE_PORT" in node_env


def test_env_example_documents_node_key_and_does_not_leak_a_real_secret():
    content = _read(".env.example")
    assert "NODE_KEY=" in content
    assert "NODE_NAME=" in content
    assert "NODE_PORT=" in content
    # 只能是空值或占位说明，不能是真实密钥
    for line in content.splitlines():
        if line.startswith("NODE_KEY="):
            assert line.strip() == "NODE_KEY="


def test_readme_documents_docker_and_register_then_inject_flow():
    content = _read("README.md")
    assert "docker build" in content
    assert "docker compose" in content or "docker-compose" in content
    assert "Dockerfile.node" in content
    assert "/api/register" in content
    assert "NODE_KEY" in content
    # 强调 key 只在注册响应中出现一次
    assert "一次" in content


def test_node_dockerfile_supports_auto_register_env_vars():
    content = _read("Dockerfile.node")
    for env_var in (
        "NODE_REGISTER_URL",
        "REGISTRATION_TOKEN",
        "NODE_ADDRESS",
        "NODE_PROTOCOL",
        "NODE_METADATA_JSON",
        "NODE_INI",
        "NODE_REGISTER_RETRIES",
    ):
        assert env_var in content
    assert "/data" in content


def test_docker_compose_web_has_healthcheck_and_node_depends_on_it():
    doc = yaml.safe_load(_read("docker-compose.yml"))
    services = doc["services"]

    web = services["web"]
    assert "healthcheck" in web

    node = services["node"]
    depends_on = node["depends_on"]
    assert depends_on["web"]["condition"] == "service_healthy"


def test_docker_compose_node_configures_auto_register_and_persists_node_ini():
    doc = yaml.safe_load(_read("docker-compose.yml"))
    services = doc["services"]
    node = services["node"]
    node_env = node.get("environment", {})

    for env_var in ("NODE_REGISTER_URL", "REGISTRATION_TOKEN", "NODE_ADDRESS", "NODE_INI"):
        assert env_var in node_env

    assert any("node-data" in v and "/data" in v for v in node.get("volumes", []))
    assert "node-data" in (doc.get("volumes") or {})


def test_env_example_documents_auto_register_vars():
    content = _read(".env.example")
    for env_var in ("NODE_REGISTER_URL=", "NODE_ADDRESS=", "NODE_INI=", "NODE_REGISTER_RETRIES="):
        assert env_var in content
    # NODE_KEY 仍然只作为高级覆盖，示例文件里不能是真实密钥。
    for line in content.splitlines():
        if line.startswith("NODE_KEY="):
            assert line.strip() == "NODE_KEY="


def test_readme_documents_auto_register_flow():
    content = _read("README.md")
    assert "node.ini" in content
    assert "自动注册" in content
    assert "/api/register/auto" in content
