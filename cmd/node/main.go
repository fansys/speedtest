// Command node 是 LibreSpeed 风格的独立节点 agent：/healthz /ping /download /upload。
//
//	go run ./cmd/node
//
// 密钥来源二选一：
//   - NODE_KEY 非空：直接使用该 key 启动，跳过自动注册。
//   - 设置 NODE_REGISTER_URL（+ REGISTRATION_TOKEN + NODE_ADDRESS + NODE_PORT +
//     NODE_INI）：启动前向中心服务自助注册，成功后把凭据原子写回 NODE_INI。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"librespeed-service/internal/config"
	"librespeed-service/internal/security"
)

func main() {
	cfg, err := config.LoadNodeConfig()
	if err != nil {
		log.Fatalf("[node-agent] 配置加载失败: %v", err)
	}

	nodeKey, err := resolveNodeKey(cfg)
	if err != nil {
		log.Fatalf("[node-agent] %v", err)
	}

	srv, err := newNodeServer(nodeKey, cfg.Name, cfg.MaxTestBytes)
	if err != nil {
		log.Fatalf("[node-agent] %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[node-agent] 监听 %s，name=%s", addr, cfg.Name)

	server := &http.Server{
		Addr:              addr,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[node-agent] 服务退出: %v", err)
	}
}

// resolveNodeKey 按优先级解析出启动要用的 node_key：显式覆盖 > 自动注册 > 报错退出。
//
// 自动注册模式下会先读取 node.ini 里上次保存的 key 作为 existing_node_key 带给
// 服务端尝试复用，成功/复用后都会把最新状态原子写回 node.ini。
func resolveNodeKey(cfg *config.NodeConfig) (string, error) {
	if key := strings.TrimSpace(cfg.NodeKey); key != "" {
		log.Printf("[node-agent] 使用显式提供的 NODE_KEY 启动（跳过自动注册），指纹=%s", security.KeyFingerprint(key))
		return key, nil
	}

	if cfg.RegisterURL == "" {
		return "", fmt.Errorf("必须通过 NODE_KEY 显式提供密钥，或设置 NODE_REGISTER_URL 启用自动注册")
	}
	if cfg.RegistrationToken == "" {
		return "", fmt.Errorf("设置了 NODE_REGISTER_URL 时必须同时提供 REGISTRATION_TOKEN")
	}
	if cfg.Address == "" {
		return "", fmt.Errorf("自动注册模式需要 NODE_ADDRESS（中心服务据此访问本节点）")
	}
	if cfg.MetadataJSON != "" {
		var probe map[string]any
		if err := json.Unmarshal([]byte(cfg.MetadataJSON), &probe); err != nil {
			return "", fmt.Errorf("NODE_METADATA_JSON 不是合法 JSON: %w", err)
		}
	}

	state, err := config.LoadNodeState(cfg.NodeIni)
	if err != nil {
		return "", err
	}
	existingKey := ""
	if state != nil {
		existingKey = strings.TrimSpace(state["node_key"])
	}

	result, err := registerWithRetries(cfg.RegisterURL, cfg.RegistrationToken, cfg.Name, cfg.Address, cfg.Protocol, cfg.Port, cfg.MetadataJSON, existingKey, cfg.RegisterRetries)
	if err != nil {
		return "", fmt.Errorf("自动注册失败，节点未启动: %w", err)
	}
	nodeKey := strings.TrimSpace(result.NodeKey)
	if nodeKey == "" {
		return "", fmt.Errorf("注册响应中没有 node_key，节点未启动")
	}

	if err := config.SaveNodeState(cfg.NodeIni, map[string]string{
		"node_id":    strconv.FormatInt(result.ID, 10),
		"node_key":   nodeKey,
		"name":       cfg.Name,
		"address":    cfg.Address,
		"port":       strconv.Itoa(cfg.Port),
		"protocol":   cfg.Protocol,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		log.Printf("[node-agent] 警告: 写入 %s 失败: %v", cfg.NodeIni, err)
	}

	statusWord := "生成了新的"
	if result.Reused {
		statusWord = "复用了已有"
	}
	log.Printf("[node-agent] 自动注册成功，%s node_key，node_id=%d 指纹=%s", statusWord, result.ID, security.KeyFingerprint(nodeKey))
	return nodeKey, nil
}
