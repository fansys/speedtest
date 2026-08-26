"""节点管理与测速 API：ADMIN_TOKEN 保护。"""

from __future__ import annotations

import json
import time

from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy.orm import Session

from ..config import Settings
from ..crypto_box import SealedDataError, unseal
from ..deps import get_db, get_settings_dep
from ..models import Node, utcnow
from ..node_client import NodeClientError, check_health, measure_download, measure_ping, measure_upload
from ..schemas import (
    HealthResult,
    NodeListOut,
    NodeOut,
    PingResult,
    SpeedtestRequest,
    SpeedtestResult,
    TransferResult,
)
from ..security import hash_node_key, require_admin

router = APIRouter(prefix="/api/nodes", tags=["nodes"], dependencies=[Depends(require_admin)])


def node_to_out(node: Node) -> NodeOut:
    metadata = json.loads(node.node_metadata) if node.node_metadata else None
    return NodeOut(
        id=node.id,
        name=node.name,
        address=node.address,
        port=node.port,
        protocol=node.protocol,
        key_fingerprint=node.key_fingerprint,
        enabled=node.enabled,
        metadata=metadata,
        created_at=node.created_at,
        updated_at=node.updated_at,
        last_seen_at=node.last_seen_at,
        last_status=node.last_status,
        last_latency_ms=node.last_latency_ms,
        last_error=node.last_error,
    )


def _get_node_or_404(db: Session, node_id: int) -> Node:
    node = db.get(Node, node_id)
    if node is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail="节点不存在")
    return node


def _resolve_node_key(node: Node, settings: Settings, supplied: str | None) -> str:
    if supplied:
        if hash_node_key(supplied) != node.node_key_hash:
            raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="node_key 与该节点不匹配")
        return supplied
    if node.node_key_sealed:
        try:
            return unseal(settings.secret_key, node.node_key_sealed)
        except SealedDataError as exc:
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail="节点密钥解封失败，请检查 SECRET_KEY 是否变更",
            ) from exc
    raise HTTPException(
        status_code=status.HTTP_400_BAD_REQUEST,
        detail="该节点未保存密钥副本（STORE_NODE_KEY_SEALED=false），请在请求体中提供 node_key",
    )


@router.get("", response_model=NodeListOut)
def list_nodes(db: Session = Depends(get_db)) -> NodeListOut:
    nodes = db.query(Node).order_by(Node.id).all()
    return NodeListOut(nodes=[node_to_out(n) for n in nodes])


@router.get("/{node_id}", response_model=NodeOut)
def get_node(node_id: int, db: Session = Depends(get_db)) -> NodeOut:
    return node_to_out(_get_node_or_404(db, node_id))


@router.post("/{node_id}/enable", response_model=NodeOut)
def enable_node(node_id: int, db: Session = Depends(get_db)) -> NodeOut:
    node = _get_node_or_404(db, node_id)
    node.enabled = True
    db.commit()
    db.refresh(node)
    return node_to_out(node)


@router.post("/{node_id}/disable", response_model=NodeOut)
def disable_node(node_id: int, db: Session = Depends(get_db)) -> NodeOut:
    node = _get_node_or_404(db, node_id)
    node.enabled = False
    db.commit()
    db.refresh(node)
    return node_to_out(node)


@router.delete("/{node_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_node(node_id: int, db: Session = Depends(get_db)) -> None:
    node = _get_node_or_404(db, node_id)
    db.delete(node)
    db.commit()


@router.post("/{node_id}/health", response_model=HealthResult)
async def health_check(
    node_id: int,
    payload: SpeedtestRequest | None = None,
    db: Session = Depends(get_db),
    settings: Settings = Depends(get_settings_dep),
) -> HealthResult:
    node = _get_node_or_404(db, node_id)
    supplied = payload.node_key if payload else None
    node_key = _resolve_node_key(node, settings, supplied)

    start = time.perf_counter()
    try:
        await check_health(node, node_key, settings)
    except NodeClientError as exc:
        node.last_status = "error"
        node.last_error = str(exc)
        db.commit()
        return HealthResult(node_id=node.id, status="error", error=str(exc))

    latency_ms = (time.perf_counter() - start) * 1000
    node.last_status = "online"
    node.last_latency_ms = latency_ms
    node.last_error = None
    node.last_seen_at = utcnow()
    db.commit()
    return HealthResult(node_id=node.id, status="online", latency_ms=latency_ms)


@router.post("/{node_id}/speedtest", response_model=SpeedtestResult)
async def speedtest(
    node_id: int,
    payload: SpeedtestRequest | None = None,
    db: Session = Depends(get_db),
    settings: Settings = Depends(get_settings_dep),
) -> SpeedtestResult:
    node = _get_node_or_404(db, node_id)
    if not node.enabled:
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail="节点已被禁用")

    supplied = payload.node_key if payload else None
    node_key = _resolve_node_key(node, settings, supplied)

    ping_count = payload.ping_count if payload else None
    download_bytes = payload.download_bytes if payload else None
    upload_bytes = payload.upload_bytes if payload else None

    result = SpeedtestResult(node_id=node.id)

    try:
        result.ping = PingResult(**await measure_ping(node, node_key, settings, ping_count))
        result.download = TransferResult(
            **await measure_download(node, node_key, settings, download_bytes)
        )
        result.upload = TransferResult(
            **await measure_upload(node, node_key, settings, upload_bytes)
        )
    except NodeClientError as exc:
        result.error = str(exc)
        node.last_status = "error"
        node.last_error = str(exc)
        db.commit()
        return result

    node.last_status = "online"
    node.last_latency_ms = result.ping.min_ms if result.ping else None
    node.last_error = None
    node.last_seen_at = utcnow()
    db.commit()
    return result
