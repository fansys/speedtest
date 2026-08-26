package main

import (
	"encoding/json"
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
// 手动注册永远不接受/复用 existing_node_key：重复注册同一 address+port 一律生成
// 新 key 并让旧 key 立即失效。
func (s *server) register(w http.ResponseWriter, r *http.Request, auto bool) {
	var req registerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求体不是合法 JSON")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusUnprocessableEntity, "name 不能为空")
		return
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = "http"
	}

	resolved, err := netguard.ValidateNodeTarget(req.Address, req.Port, protocol, netguard.ValidateNodeTargetOptions{
		AllowPrivate:     s.settings.AllowPrivateNodes,
		AllowedProtocols: s.settings.AllowedNodeProtocols,
		Resolve:          true,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var metadataJSON *string
	if len(req.Metadata) > 0 {
		b, err := json.Marshal(req.Metadata)
		if err != nil {
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
		writeError(w, http.StatusInternalServerError, "注册失败: "+err.Error())
		return
	}

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
