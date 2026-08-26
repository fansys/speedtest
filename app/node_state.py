"""node.ini 本地持久化：保存节点自助注册后拿到的 node_id / node_key 等状态。

写入采用「临时文件 + os.replace 原子替换」，容器被杀掉/断电等异常情况下不会
留下半写的 node.ini；文件权限尽量收紧到 0600（仅当前用户可读写）。
"""

from __future__ import annotations

import configparser
import os
import tempfile
from pathlib import Path
from typing import Any

SECTION = "node"


def load_node_state(path: str | os.PathLike[str]) -> dict[str, str] | None:
    """读取 node.ini；文件不存在或没有 [node] 段时返回 None。"""
    p = Path(path)
    if not p.is_file():
        return None
    parser = configparser.ConfigParser()
    parser.read(p, encoding="utf-8")
    if SECTION not in parser:
        return None
    return dict(parser[SECTION])


def save_node_state(path: str | os.PathLike[str], **fields: Any) -> None:
    """原子写入 node.ini，覆盖 [node] 段为 ``fields``（None 值会被跳过）。"""
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)

    parser = configparser.ConfigParser()
    parser[SECTION] = {k: str(v) for k, v in fields.items() if v is not None}

    fd, tmp_name = tempfile.mkstemp(dir=str(p.parent), prefix=".node-ini-", suffix=".tmp")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            parser.write(f)
        try:
            os.chmod(tmp_name, 0o600)
        except OSError:
            pass  # 部分文件系统（如某些挂载卷）不支持 chmod，尽力而为即可
        os.replace(tmp_name, p)
    except BaseException:
        try:
            os.unlink(tmp_name)
        except OSError:
            pass
        raise

    try:
        os.chmod(p, 0o600)
    except OSError:
        pass
