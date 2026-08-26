// Package config 提供中心 Web 服务与节点 agent 的集中配置。
//
// 所有敏感值都只从环境变量 / .env 读取，绝不写入日志或 API 响应。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MinTokenLength 是 ADMIN_TOKEN / REGISTRATION_TOKEN / SECRET_KEY 的最小长度要求。
const MinTokenLength = 16

// Settings 是中心 Web 服务的配置。
type Settings struct {
	// ---- 鉴权 ----
	AdminToken        string
	RegistrationToken string
	SecretKey         string

	// ---- 服务 ----
	Host        string
	Port        int
	DatabaseURL string
	LogLevel    string

	// ---- 节点访问 / SSRF 控制 ----
	AllowPrivateNodes    bool
	AllowedNodeProtocols []string
	NodeConnectTimeout   float64
	NodeRequestTimeout   float64
	NodeHealthTimeout    float64

	// ---- 测速参数 ----
	MaxTestBytes           int64
	DefaultDownloadBytes   int64
	DefaultUploadBytes     int64
	StreamChunkBytes       int
	PingCount              int
	EnableCentralSpeedtest bool // 控制中心 Web 节点自身是否开启测速功能 (ENABLE_CENTRAL_SPEEDTEST / ENABLE_WEB_SPEEDTEST, 默认 true)

	// ---- 节点 key 存储策略 ----
	StoreNodeKeySealed bool
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s 不是合法的布尔值: %q", key, v)
	}
	return parsed, nil
}

func getEnvInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法的整数: %q", key, v)
	}
	return parsed, nil
}

func getEnvInt64(key string, def int64) (int64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	parsed, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法的整数: %q", key, v)
	}
	return parsed, nil
}

func getEnvFloat(key string, def float64) (float64, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def, nil
	}
	parsed, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%s 不是合法的浮点数: %q", key, v)
	}
	return parsed, nil
}

// Load 从环境变量（.env 作为兜底）加载并校验 Settings。
func Load() (*Settings, error) {
	loadDotEnv(".env")

	s := &Settings{
		AdminToken:        getEnv("ADMIN_TOKEN", ""),
		RegistrationToken: getEnv("REGISTRATION_TOKEN", ""),
		SecretKey:         getEnv("SECRET_KEY", ""),
		Host:              getEnv("HOST", "127.0.0.1"),
		DatabaseURL:       getEnv("DATABASE_URL", "sqlite:./data/librespeed.db"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
	}

	var err error
	if s.Port, err = getEnvInt("PORT", 8080); err != nil {
		return nil, err
	}
	if s.AllowPrivateNodes, err = getEnvBool("ALLOW_PRIVATE_NODES", true); err != nil {
		return nil, err
	}

	protocolsRaw := getEnv("ALLOWED_NODE_PROTOCOLS", "http,https")
	protocols, err := normalizeProtocols(protocolsRaw)
	if err != nil {
		return nil, err
	}
	s.AllowedNodeProtocols = protocols

	if s.NodeConnectTimeout, err = getEnvFloat("NODE_CONNECT_TIMEOUT", 5.0); err != nil {
		return nil, err
	}
	if s.NodeRequestTimeout, err = getEnvFloat("NODE_REQUEST_TIMEOUT", 30.0); err != nil {
		return nil, err
	}
	if s.NodeHealthTimeout, err = getEnvFloat("NODE_HEALTH_TIMEOUT", 5.0); err != nil {
		return nil, err
	}

	if s.MaxTestBytes, err = getEnvInt64("MAX_TEST_BYTES", 512*1024*1024); err != nil {
		return nil, err
	}
	if s.DefaultDownloadBytes, err = getEnvInt64("DEFAULT_DOWNLOAD_BYTES", 32*1024*1024); err != nil {
		return nil, err
	}
	if s.DefaultUploadBytes, err = getEnvInt64("DEFAULT_UPLOAD_BYTES", 16*1024*1024); err != nil {
		return nil, err
	}
	streamChunk, err := getEnvInt("STREAM_CHUNK_BYTES", 64*1024)
	if err != nil {
		return nil, err
	}
	s.StreamChunkBytes = streamChunk
	if s.PingCount, err = getEnvInt("PING_COUNT", 6); err != nil {
		return nil, err
	}
	if s.StoreNodeKeySealed, err = getEnvBool("STORE_NODE_KEY_SEALED", true); err != nil {
		return nil, err
	}

	// 控制中心 Web 节点自身是否开启测速功能 (默认 true，可通过 ENABLE_CENTRAL_SPEEDTEST=false 关闭)
	if s.EnableCentralSpeedtest, err = getEnvBool("ENABLE_CENTRAL_SPEEDTEST", true); err != nil {
		return nil, err
	}
	if v, ok := os.LookupEnv("ENABLE_WEB_SPEEDTEST"); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			s.EnableCentralSpeedtest = b
		}
	}

	if err := s.validate(); err != nil {
		return nil, err
	}
	return s, nil
}

func normalizeProtocols(raw string) ([]string, error) {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if p != "http" && p != "https" {
			return nil, fmt.Errorf("ALLOWED_NODE_PROTOCOLS 只支持 http/https，收到: %q", p)
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ALLOWED_NODE_PROTOCOLS 不能为空")
	}
	return out, nil
}

func (s *Settings) validate() error {
	for name, value := range map[string]string{
		"ADMIN_TOKEN":        s.AdminToken,
		"REGISTRATION_TOKEN": s.RegistrationToken,
	} {
		if value == "" {
			return fmt.Errorf("%s 未设置。请在 .env 中配置，可用 `openssl rand -base64 32` 生成", name)
		}
		if len(value) < MinTokenLength {
			return fmt.Errorf("%s 长度至少 %d 个字符", name, MinTokenLength)
		}
	}
	if s.AdminToken == s.RegistrationToken {
		return fmt.Errorf("ADMIN_TOKEN 与 REGISTRATION_TOKEN 不能相同")
	}
	if s.StoreNodeKeySealed && len(s.SecretKey) < MinTokenLength {
		return fmt.Errorf("启用 STORE_NODE_KEY_SEALED 时必须设置长度 >= %d 的 SECRET_KEY", MinTokenLength)
	}
	return nil
}

// SecretValues 返回所有需要在日志中屏蔽的敏感值。
func (s *Settings) SecretValues() []string {
	var out []string
	for _, v := range []string{s.AdminToken, s.RegistrationToken, s.SecretKey} {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// DatabasePath 把 DATABASE_URL 归一化成文件系统路径。
func (s *Settings) DatabasePath() string {
	v := s.DatabaseURL
	switch {
	case strings.HasPrefix(v, "sqlite:///"):
		return "/" + strings.TrimPrefix(v, "sqlite:///")
	case strings.HasPrefix(v, "sqlite://"):
		return strings.TrimPrefix(v, "sqlite://")
	case strings.HasPrefix(v, "sqlite:"):
		return strings.TrimPrefix(v, "sqlite:")
	default:
		return v
	}
}
