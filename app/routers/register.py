"""节点注册 API：REGISTRATION_TOKEN 保护。"""

from __future__ import annotations

import json

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session

from ..config import Settings
from ..crypto_box import seal
from ..deps import get_db, get_settings_dep
from ..models import Node
from ..netguard import NodeAddressError, validate_node_target
from ..schemas import AutoRegisterOut, AutoRegisterRequest, RegisterOut, RegisterRequest
from ..security import generate_node_key, hash_node_key, key_fingerprint, require_registration
from .nodes import node_to_out

router = APIRouter(tags=["register"])


def _register_or_reuse(
    *,
    name: str,
    address: str,
    port: int,
    protocol: str,
    metadata: dict | None,
    existing_node_key: str | None,
    db: Session,
    settings: Settings,
) -> tuple[Node, str, bool]:
    """核心注册逻辑，供手动 ``/api/register`` 与自助 ``/api/register/auto`` 复用。

    以 address+port 作为节点身份。``existing_node_key`` 只有在它确实属于「当前
    这个 address+port 对应的节点」时才会被复用（不轮换）；否则一律生成新 key
    并（视情况）创建或轮换该 address+port 上的节点——绝不会去改动 key 实际
    归属的另一个节点，防止旧 key 被用来冒领/覆盖别的地址端口。
    """
    try:
        resolved = validate_node_target(
            address,
            port,
            protocol,
            allow_private=settings.allow_private_nodes,
            allowed_protocols=settings.protocol_list,
            resolve=True,
        )
    except NodeAddressError as exc:
        raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail=str(exc)) from exc

    target = db.query(Node).filter(Node.address == resolved.host, Node.port == resolved.port).one_or_none()
    metadata_json = json.dumps(metadata, ensure_ascii=False) if metadata else None

    if existing_node_key:
        owner = db.query(Node).filter(Node.node_key_hash == hash_node_key(existing_node_key)).one_or_none()
        if owner is not None and target is not None and owner.id == target.id:
            target.name = name
            target.protocol = resolved.protocol
            target.node_metadata = metadata_json
            target.enabled = True
            db.commit()
            db.refresh(target)
            return target, existing_node_key, True

    node_key = generate_node_key()
    node_hash = hash_node_key(node_key)
    fingerprint = key_fingerprint(node_key)
    sealed = seal(settings.secret_key, node_key) if settings.store_node_key_sealed else None

    if target is None:
        target = Node(
            name=name,
            address=resolved.host,
            port=resolved.port,
            protocol=resolved.protocol,
            node_key_hash=node_hash,
            key_fingerprint=fingerprint,
            node_key_sealed=sealed,
            node_metadata=metadata_json,
            enabled=True,
        )
        db.add(target)
    else:
        target.name = name
        target.protocol = resolved.protocol
        target.node_key_hash = node_hash
        target.key_fingerprint = fingerprint
        target.node_key_sealed = sealed
        target.node_metadata = metadata_json
        target.enabled = True

    db.commit()
    db.refresh(target)
    return target, node_key, False


@router.post(
    "/api/register",
    response_model=RegisterOut,
    dependencies=[Depends(require_registration)],
)
def register_node(
    payload: RegisterRequest,
    db: Session = Depends(get_db),
    settings: Settings = Depends(get_settings_dep),
) -> RegisterOut:
    # 手动注册不接受也不复用 key：重复注册同一地址/端口一律生成新 key 并
    # 让旧 key 立即失效（沿用既有行为，不受下面自动注册的复用逻辑影响）。
    node, node_key, _reused = _register_or_reuse(
        name=payload.name,
        address=payload.address,
        port=payload.port,
        protocol=payload.protocol,
        metadata=payload.metadata,
        existing_node_key=None,
        db=db,
        settings=settings,
    )
    return RegisterOut(**node_to_out(node).model_dump(), node_key=node_key)


@router.post(
    "/api/register/auto",
    response_model=AutoRegisterOut,
    dependencies=[Depends(require_registration)],
)
def register_node_auto(
    payload: AutoRegisterRequest,
    db: Session = Depends(get_db),
    settings: Settings = Depends(get_settings_dep),
) -> AutoRegisterOut:
    """供节点 agent 启动时调用的自助注册接口。

    明文 node_key 只会出现在这个受 REGISTRATION_TOKEN 保护的响应里；
    普通管理 API（``/api/nodes/*``）永远不会返回它。
    """
    node, node_key, reused = _register_or_reuse(
        name=payload.name,
        address=payload.address,
        port=payload.port,
        protocol=payload.protocol,
        metadata=payload.metadata,
        existing_node_key=payload.existing_node_key,
        db=db,
        settings=settings,
    )
    return AutoRegisterOut(**node_to_out(node).model_dump(), node_key=node_key, reused=reused)
