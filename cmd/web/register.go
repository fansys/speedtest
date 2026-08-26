package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"librespeed-service/internal/netguard"
	"librespeed-service/internal/sealedbox"
	"librespeed-service/internal/security"
	"librespeed-service/internal/store"
)

func (s *server) handleRegister(w http.ResponseWriter, r *http.Request) {
	s.register(w, r, false)
}

func (s *server) handleRegisterAuto(w http.ResponseWriter, r *http.Request) {
	s.register(w, r, true)
}

// register 是手动 /api/register 与自助 /api/register/auto 共用的核心逻辑。
func (s *server) register(w http.ResponseWriter, r *http.Request, auto bool) {
	var req registerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		log.Printf("[web] 节点注册请求失败: 请求体非法 JSON 或超出大小限制: %v, 来源IP=%s", err, r.RemoteAddr)
		writeError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		log.Printf("[web] 节点注册请求失败: name 为空, 来源IP=%s", r.RemoteAddr)
		writeError(w, http.StatusUnprocessableEntity, "name 不能为空")
		return
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = "http"
	}

	hasExistingKey := req.ExistingNodeKey != nil && *req.ExistingNodeKey != ""
	existingFp := ""
	if hasExistingKey {
		existingFp = security.KeyFingerprint(*req.ExistingNodeKey)
	}

	log.Printf("[web] 收到节点注册请求: auto=%t, name=%q, address=%s:%d, proto=%s, 携带旧Key=%t(fp=%s), 来源IP=%s",
		auto, req.Name, req.Address, req.Port, protocol, hasExistingKey, existingFp, r.RemoteAddr)

	resolved, err := netguard.ValidateNodeTarget(req.Address, req.Port, protocol, netguard.ValidateNodeTargetOptions{
		AllowPrivate:     s.settings.AllowPrivateNodes,
		AllowedProtocols: s.settings.AllowedNodeProtocols,
		Resolve:          true,
	})
	if err != nil {
		log.Printf("[web] 节点注册安全拦截: 目标地址 %s:%d 被 netguard 拒绝: %v, 来源IP=%s", req.Address, req.Port, err, r.RemoteAddr)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var metadataJSON *string
	if len(req.Metadata) > 0 {
		b, err := json.Marshal(req.Metadata)
		if err != nil {
			log.Printf("[web] 节点注册失败: metadata JSON 序列化失败: %v", err)
			writeError(w, http.StatusBadRequest, "metadata 不是合法 JSON")
			return
		}
		v := string(b)
		metadataJSON = &v
	}

	var existingKey *string
	if auto && req.ExistingNodeKey != nil && *req.ExistingNodeKey != "" {
		existingKey = req.ExistingNodeKey
	}

	params := store.RegisterParams{
		Name:            req.Name,
		Address:         resolved.Host,
		Port:            resolved.Port,
		Protocol:        resolved.Protocol,
		MetadataJSON:    metadataJSON,
		ExistingNodeKey: existingKey,
		GenerateKey:     security.GenerateNodeKey,
		HashKey:         security.HashNodeKey,
		Fingerprint:     security.KeyFingerprint,
		SealKey: func(nodeKey string) (*string, error) {
			if !s.settings.StoreNodeKeySealed {
				return nil, nil
			}
			sealed, err := sealedbox.Seal(s.settings.SecretKey, nodeKey)
			if err != nil {
				return nil, err
			}
			return &sealed, nil
		},
	}

	node, nodeKey, reused, err := s.store.RegisterOrReuse(params)
	if err != nil {
		log.Printf("[web] 节点注册持久化失败: %v, name=%q target=%s:%d", err, req.Name, resolved.Host, resolved.Port)
		writeError(w, http.StatusInternalServerError, "注册失败: "+err.Error())
		return
	}

	statusWord := "新生成Key"
	if reused {
		statusWord = "复用旧Key"
	}
	log.Printf("[web] 节点注册成功: id=%d, name=%q, 测速端点=%s://%s:%d, Key状态=%s, Key指纹=%s, auto=%t",
		node.ID, node.Name, node.Protocol, node.Address, node.Port, statusWord, node.KeyFingerprint, auto)

	out, err := nodeToOut(node)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "序列化失败")
		return
	}

	if auto {
		writeJSON(w, http.StatusOK, autoRegisterOut{registerOut{nodeOut: out, NodeKey: nodeKey}, reused})
		return
	}
	writeJSON(w, http.StatusOK, registerOut{nodeOut: out, NodeKey: nodeKey})
}
