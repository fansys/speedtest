// Command node 是 LibreSpeed 风格的独立节点 agent：/healthz /ping /download /upload。
//
//	go run ./cmd/node
//
// 端口支持分别配置：
//   - 本地绑定监听端口：由 NODE_LISTEN_PORT (或 NODE_PORT) 指定，默认 8081；
//   - 外部公布测速端口：由 NODE_ADVERTISE_PORT (或 NODE_PORT) 指定，默认等于监听端口。
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

	log.Printf("[node-agent] 启动准备: name=%q, host=%s, 本地监听端口=%d, 外部测速端口=%d, 协议=%s",
		cfg.Name, cfg.Host, cfg.ListenPort, cfg.Port(), cfg.Protocol)

	nodeKey, err := resolveNodeKey(cfg)
	if err != nil {
		log.Fatalf("[node-agent] 密钥初始化失败，节点退出: %v", err)
	}

	srv, err := newNodeServer(nodeKey, cfg.Name, cfg.MaxTestBytes)
	if err != nil {
		log.Fatalf("[node-agent] 创建服务失败: %v", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.ListenPort)
	log.Printf("[node-agent] 服务已就绪，正在监听 %s (向中心公布的外部测速端口: %d), 密钥指纹=%s",
		addr, cfg.Port(), security.KeyFingerprint(nodeKey))

	server := &http.Server{
		Addr:              addr,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[node-agent] 服务异常退出: %v", err)
	}
}

// resolveNodeKey 按优先级解析出启动要用的 node_key：显式覆盖 > 自动注册 > 报错退出。
func resolveNodeKey(cfg *config.NodeConfig) (string, error) {
	if key := strings.TrimSpace(cfg.NodeKey); key != "" {
		log.Printf("[node-agent] 使用显式环境变量 NODE_KEY 启动（跳过自动注册），指纹=%s", security.KeyFingerprint(key))
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
		log.Printf("[node-agent] 读取本地历史状态文件 %s 失败: %v", cfg.NodeIni, err)
		return "", err
	}
	existingKey := ""
	if state != nil {
		existingKey = strings.TrimSpace(state["node_key"])
		log.Printf("[node-agent] 已加载本地历史凭据 (%s): node_id=%s, 旧Key指纹=%s",
			cfg.NodeIni, state["node_id"], security.KeyFingerprint(existingKey))
	} else {
		log.Printf("[node-agent] 本地状态文件 %s 不存在，将以首次注册模式启动", cfg.NodeIni)
	}

	result, err := registerWithRetries(
		cfg.RegisterURL,
		cfg.RegistrationToken,
		cfg.Name,
		cfg.Address,
		cfg.Protocol,
		cfg.Port(), // 向中心注册外部公布的测速端口
		cfg.MetadataJSON,
		existingKey,
		cfg.RegisterRetries,
	)
	if err != nil {
		return "", fmt.Errorf("自动注册失败，节点拒绝启动: %w", err)
	}
	nodeKey := strings.TrimSpace(result.NodeKey)
	if nodeKey == "" {
		return "", fmt.Errorf("注册响应中未包含 node_key，节点拒绝启动")
	}

	// 原子写回 node.ini (0600 权限)
	if err := config.SaveNodeState(cfg.NodeIni, map[string]string{
		"node_id":     strconv.FormatInt(result.ID, 10),
		"node_key":    nodeKey,
		"name":        cfg.Name,
		"address":     cfg.Address,
		"port":        strconv.Itoa(cfg.Port()),
		"listen_port": strconv.Itoa(cfg.ListenPort),
		"protocol":    cfg.Protocol,
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		log.Printf("[node-agent] 警告: 写入凭据到 %s 失败: %v", cfg.NodeIni, err)
	} else {
		log.Printf("[node-agent] 注册凭据已原子持久化到 %s (权限 0600)", cfg.NodeIni)
	}

	statusWord := "新生成Key"
	if result.Reused {
		statusWord = "复用已有Key"
	}
	log.Printf("[node-agent] 节点注册就绪: node_id=%d, 测速路由=%s://%s:%d, Key状态=%s, Key指纹=%s",
		result.ID, cfg.Protocol, cfg.Address, cfg.Port(), statusWord, security.KeyFingerprint(nodeKey))
	return nodeKey, nil
}
