package config

import (
	"fmt"
	"os"
	"strconv"
)

// NodeConfig 是节点 agent 的配置，直接从环境变量读取。
// 支持分别配置本地监听绑定端口 (ListenPort) 与外部公布测速端口 (AdvertisePort)。
type NodeConfig struct {
	Host          string
	ListenPort    int // 本地 HTTP 监听绑定的端口（默认 8081，由 NODE_LISTEN_PORT 或 NODE_PORT 控制）
	AdvertisePort int // 注册给中心服务/外部访问的测速端口（默认等于 ListenPort，由 NODE_ADVERTISE_PORT 或 NODE_PORT 控制）
	Name          string

	// NodeKey 非空时跳过自动注册，直接用这个 key 启动。
	NodeKey string

	// 自动注册相关配置。
	RegisterURL       string
	RegistrationToken string
	Address           string
	Protocol          string
	MetadataJSON      string
	NodeIni           string
	RegisterRetries   int

	MaxTestBytes int64
	LogLevel     string
}

// Port 返回向中心注册的测速端口（等于 AdvertisePort）。
func (c *NodeConfig) Port() int {
	if c.AdvertisePort > 0 {
		return c.AdvertisePort
	}
	return c.ListenPort
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// LoadNodeConfig 从环境变量加载节点 agent 配置。
func LoadNodeConfig() (*NodeConfig, error) {
	c := &NodeConfig{
		Host:              envOr("NODE_HOST", "0.0.0.0"),
		Name:              envOr("NODE_NAME", "node"),
		NodeKey:           os.Getenv("NODE_KEY"),
		RegisterURL:       firstNonEmpty(os.Getenv("NODE_REGISTER_URL"), os.Getenv("REGISTER_URL")),
		RegistrationToken: os.Getenv("REGISTRATION_TOKEN"),
		Address:           os.Getenv("NODE_ADDRESS"),
		Protocol:          envOr("NODE_PROTOCOL", "http"),
		MetadataJSON:      os.Getenv("NODE_METADATA_JSON"),
		NodeIni:           envOr("NODE_INI", "./node.ini"),
		LogLevel:          envOr("NODE_LOG_LEVEL", "info"),
	}

	// 解析本地监听端口 (NODE_LISTEN_PORT > NODE_PORT > 8081)
	listenPortStr := firstNonEmpty(os.Getenv("NODE_LISTEN_PORT"), os.Getenv("NODE_PORT"), "8081")
	listenPort, err := strconv.Atoi(listenPortStr)
	if err != nil || listenPort < 1 || listenPort > 65535 {
		return nil, fmt.Errorf("NODE_LISTEN_PORT / NODE_PORT 不是合法的端口 (1-65535): %q", listenPortStr)
	}
	c.ListenPort = listenPort

	// 解析外部公布测速端口 (NODE_ADVERTISE_PORT > NODE_PORT > ListenPort)
	advPortStr := firstNonEmpty(os.Getenv("NODE_ADVERTISE_PORT"), os.Getenv("NODE_PORT"), listenPortStr)
	advPort, err := strconv.Atoi(advPortStr)
	if err != nil || advPort < 1 || advPort > 65535 {
		return nil, fmt.Errorf("NODE_ADVERTISE_PORT / NODE_PORT 不是合法的端口 (1-65535): %q", advPortStr)
	}
	c.AdvertisePort = advPort

	retries, err := strconv.Atoi(envOr("NODE_REGISTER_RETRIES", "10"))
	if err != nil {
		return nil, fmt.Errorf("NODE_REGISTER_RETRIES 不是合法的整数: %w", err)
	}
	c.RegisterRetries = retries

	maxBytes, err := strconv.ParseInt(envOr("MAX_TEST_BYTES", strconv.Itoa(512*1024*1024)), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("MAX_TEST_BYTES 不是合法的整数: %w", err)
	}
	c.MaxTestBytes = maxBytes

	return c, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
