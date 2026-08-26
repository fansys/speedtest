package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"librespeed-service/internal/sealedbox"
	"librespeed-service/internal/security"
	"librespeed-service/internal/store"
)

func (s *server) parseNodeID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

func (s *server) getNodeOr404(w http.ResponseWriter, r *http.Request) (*store.Node, bool) {
	id, ok := s.parseNodeID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "非法的节点 id")
		return nil, false
	}
	node, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "节点不存在")
		} else {
			writeError(w, http.StatusInternalServerError, "查询失败")
		}
		return nil, false
	}
	return node, true
}

func (s *server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "查询失败")
		return
	}
	outs := make([]nodeOut, 0, len(nodes))
	for _, n := range nodes {
		out, err := nodeToOut(n)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "序列化失败")
			return
		}
		outs = append(outs, out)
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": outs})
}

func (s *server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	node, ok := s.getNodeOr404(w, r)
	if !ok {
		return
	}
	out, err := nodeToOut(node)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "序列化失败")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleEnableNode(w http.ResponseWriter, r *http.Request) {
	s.setEnabled(w, r, true)
}

func (s *server) handleDisableNode(w http.ResponseWriter, r *http.Request) {
	s.setEnabled(w, r, false)
}

func (s *server) setEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	id, ok := s.parseNodeID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "非法的节点 id")
		return
	}
	node, err := s.store.SetEnabled(id, enabled)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "节点不存在")
		} else {
			writeError(w, http.StatusInternalServerError, "更新失败")
		}
		return
	}
	out, err := nodeToOut(node)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "序列化失败")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id, ok := s.parseNodeID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "非法的节点 id")
		return
	}
	if err := s.store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "节点不存在")
		} else {
			writeError(w, http.StatusInternalServerError, "删除失败")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resolveNodeKey 决定这次调用要用哪个 node_key：请求体/请求头显式提供的优先；
// 否则尝试用 SECRET_KEY 解封存储的密文；都没有则报错。
func (s *server) resolveNodeKey(node *store.Node, supplied *string) (nodeKey string, status int, detail string) {
	if supplied != nil && *supplied != "" {
		if security.HashNodeKey(*supplied) != node.NodeKeyHash {
			return "", http.StatusBadRequest, "node_key 与该节点不匹配"
		}
		return *supplied, 0, ""
	}
	if node.NodeKeySealed != nil {
		key, err := sealedbox.Unseal(s.settings.SecretKey, *node.NodeKeySealed)
		if err != nil {
			return "", http.StatusInternalServerError, "节点密钥解封失败，请检查 SECRET_KEY 是否变更"
		}
		return key, 0, ""
	}
	return "", http.StatusBadRequest, "该节点未保存密钥副本（STORE_NODE_KEY_SEALED=false），请在请求中提供 node_key"
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if r.ContentLength == 0 {
		return nil
	}
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v)
}

func (s *server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	node, ok := s.getNodeOr404(w, r)
	if !ok {
		return
	}

	var req speedtestRequest
	if err := decodeOptionalJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}

	headerKey := r.Header.Get("X-Node-Key")
	if req.NodeKey == nil && headerKey != "" {
		req.NodeKey = &headerKey
	}

	nodeKey, status, detail := s.resolveNodeKey(node, req.NodeKey)
	if status != 0 {
		writeError(w, status, detail)
		return
	}

	start := time.Now()
	if err := s.checkHealth(node, nodeKey); err != nil {
		msg := err.Error()
		_ = s.store.UpdateHealthResult(node.ID, "error", nil, &msg, false)
		writeJSON(w, http.StatusOK, healthResult{NodeID: node.ID, Status: "error", Error: &msg})
		return
	}
	latencyMs := time.Since(start).Seconds() * 1000
	_ = s.store.UpdateHealthResult(node.ID, "online", &latencyMs, nil, true)
	writeJSON(w, http.StatusOK, healthResult{NodeID: node.ID, Status: "online", LatencyMs: &latencyMs})
}

func (s *server) handleSpeedtest(w http.ResponseWriter, r *http.Request) {
	node, ok := s.getNodeOr404(w, r)
	if !ok {
		return
	}
	if !node.Enabled {
		writeError(w, http.StatusConflict, "节点已被禁用")
		return
	}

	var req speedtestRequest
	if err := decodeOptionalJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}

	headerKey := r.Header.Get("X-Node-Key")
	if req.NodeKey == nil && headerKey != "" {
		req.NodeKey = &headerKey
	}

	nodeKey, status, detail := s.resolveNodeKey(node, req.NodeKey)
	if status != 0 {
		writeError(w, status, detail)
		return
	}

	result := speedtestResult{NodeID: node.ID}
	err := func() error {
		pingCount := 0
		if req.PingCount != nil {
			pingCount = *req.PingCount
		}
		ping, err := s.measurePing(node, nodeKey, pingCount)
		if err != nil {
			return err
		}
		result.Ping = &ping

		var downloadBytes int64
		if req.DownloadBytes != nil {
			downloadBytes = *req.DownloadBytes
		}
		download, err := s.measureDownload(node, nodeKey, downloadBytes)
		if err != nil {
			return err
		}
		result.Download = &download

		var uploadBytes int64
		if req.UploadBytes != nil {
			uploadBytes = *req.UploadBytes
		}
		upload, err := s.measureUpload(node, nodeKey, uploadBytes)
		if err != nil {
			return err
		}
		result.Upload = &upload
		return nil
	}()

	if err != nil {
		msg := err.Error()
		result.Error = &msg
		_ = s.store.UpdateHealthResult(node.ID, "error", nil, &msg, false)
		writeJSON(w, http.StatusOK, result)
		return
	}

	latency := result.Ping.MinMs
	_ = s.store.UpdateHealthResult(node.ID, "online", &latency, nil, true)
	writeJSON(w, http.StatusOK, result)
}

// handleNodePingProxy 代理浏览器对节点的即时 Ping
func (s *server) handleNodePingProxy(w http.ResponseWriter, r *http.Request) {
	node, ok := s.getNodeOr404(w, r)
	if !ok {
		return
	}
	if !node.Enabled {
		writeError(w, http.StatusConflict, "节点已被禁用")
		return
	}

	headerKey := r.Header.Get("X-Node-Key")
	var keyPtr *string
	if headerKey != "" {
		keyPtr = &headerKey
	}
	nodeKey, status, detail := s.resolveNodeKey(node, keyPtr)
	if status != 0 {
		writeError(w, status, detail)
		return
	}

	base, err := s.nodeBaseURL(node)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeConnectTimeout)}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, base+"/ping", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建请求失败")
		return
	}
	req.Header.Set("X-Node-Key", nodeKey)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Ping 失败: %v", err))
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		writeError(w, http.StatusBadGateway, "节点拒绝了 node_key (401)")
		return
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("节点返回异常状态码 %d", resp.StatusCode))
		return
	}

	latencyMs := time.Since(start).Seconds() * 1000
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]any{
		"latency_ms": latencyMs,
	})
}

// handleNodeDownloadProxy 代理浏览器对节点的实时流式下载测速
func (s *server) handleNodeDownloadProxy(w http.ResponseWriter, r *http.Request) {
	node, ok := s.getNodeOr404(w, r)
	if !ok {
		return
	}
	if !node.Enabled {
		writeError(w, http.StatusConflict, "节点已被禁用")
		return
	}

	headerKey := r.Header.Get("X-Node-Key")
	var keyPtr *string
	if headerKey != "" {
		keyPtr = &headerKey
	}
	nodeKey, status, detail := s.resolveNodeKey(node, keyPtr)
	if status != 0 {
		writeError(w, status, detail)
		return
	}

	base, err := s.nodeBaseURL(node)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	bytesQuery := r.URL.Query().Get("bytes")
	if bytesQuery == "" {
		bytesQuery = strconv.FormatInt(s.settings.DefaultDownloadBytes, 10)
	}

	nodeURL := base + "/download?" + url.Values{"bytes": {bytesQuery}}.Encode()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, nodeURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建请求失败")
		return
	}
	req.Header.Set("X-Node-Key", nodeKey)

	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeRequestTimeout)}
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("下载测速连接节点失败: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("节点返回异常状态码 %d", resp.StatusCode))
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		w.Header().Set("Content-Length", cl)
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusOK)

	_, _ = io.Copy(w, resp.Body)
}

// handleNodeUploadProxy 代理浏览器对节点的实时流式上传测速
func (s *server) handleNodeUploadProxy(w http.ResponseWriter, r *http.Request) {
	node, ok := s.getNodeOr404(w, r)
	if !ok {
		return
	}
	if !node.Enabled {
		writeError(w, http.StatusConflict, "节点已被禁用")
		return
	}

	headerKey := r.Header.Get("X-Node-Key")
	var keyPtr *string
	if headerKey != "" {
		keyPtr = &headerKey
	}
	nodeKey, status, detail := s.resolveNodeKey(node, keyPtr)
	if status != 0 {
		writeError(w, status, detail)
		return
	}

	base, err := s.nodeBaseURL(node)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	limitedBody := io.LimitReader(r.Body, s.settings.MaxTestBytes+1)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, base+"/upload", limitedBody)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建请求失败")
		return
	}
	req.Header.Set("X-Node-Key", nodeKey)
	if r.ContentLength > 0 {
		req.ContentLength = r.ContentLength
	}

	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeRequestTimeout)}
	resp, err := client.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("上传测速连接节点失败: %v", err))
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
