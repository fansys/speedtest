package config

import (
	"fmt"
	"os"
	"strconv"
)

// NodeConfig 是节点 agent 的配置，直接从环境变量读取（与 docker-compose / 手动启动
// 传入的环境变量一一对应）。
type NodeConfig struct {
	Host string
	Port int
	Name string

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

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// LoadNodeConfig 从环境变量加载节点 agent 配置（不做业务校验，校验放在
// registration.ResolveNodeKey / cmd/node 里，方便返回更具体的错误信息）。
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

	port, err := strconv.Atoi(envOr("NODE_PORT", "8081"))
	if err != nil {
		return nil, fmt.Errorf("NODE_PORT 不是合法的整数: %w", err)
	}
	c.Port = port

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
