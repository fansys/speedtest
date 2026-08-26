package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"librespeed-service/internal/security"
)

// registerResult 是中心服务 /api/register/auto 响应里我们关心的字段。
type registerResult struct {
	ID      int64  `json:"id"`
	NodeKey string `json:"node_key"`
	Reused  bool   `json:"reused"`
}

const (
	registerTimeout  = 10 * time.Second
	registerMaxDelay = 16 * time.Second
)

// registerOnce 发起一次自助注册请求，失败返回 error（绝不泄露 REGISTRATION_TOKEN 或完整 Key）。
func registerOnce(registerURL, token, name, address, protocol string, port int, metadataJSON, existingKey string) (*registerResult, error) {
	u, err := url.Parse(registerURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("NODE_REGISTER_URL 只允许 http/https 且必须是合法 URL: %q", registerURL)
	}

	payload := map[string]any{
		"name":     name,
		"address":  address,
		"port":     port,
		"protocol": protocol,
	}
	if metadataJSON != "" {
		var m map[string]any
		if err := json.Unmarshal([]byte(metadataJSON), &m); err != nil {
			return nil, fmt.Errorf("NODE_METADATA_JSON 不是合法 JSON: %w", err)
		}
		payload["metadata"] = m
	}
	if existingKey != "" {
		payload["existing_node_key"] = existingKey
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, registerURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Registration-Token", token)

	client := &http.Client{Timeout: registerTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("注册网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errData map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errData)
		detail := ""
		if msg, ok := errData["detail"].(string); ok {
			detail = fmt.Sprintf(" (%s)", msg)
		}
		return nil, fmt.Errorf("中心服务拒绝注册: HTTP %d%s", resp.StatusCode, detail)
	}

	var out registerResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("注册响应 JSON 解析失败: %w", err)
	}
	return &out, nil
}

// registerWithRetries 指数退避重试注册：1、2、4、8、16... 秒（封顶 registerMaxDelay）。
func registerWithRetries(registerURL, token, name, address, protocol string, port int, metadataJSON, existingKey string, retries int) (*registerResult, error) {
	if retries < 1 {
		retries = 1
	}

	hasKey := existingKey != ""
	keyFp := ""
	if hasKey {
		keyFp = security.KeyFingerprint(existingKey)
	}

	var lastErr error
	delay := time.Second

	for attempt := 1; attempt <= retries; attempt++ {
		log.Printf("[node-agent] 正在向中心注册 (尝试 %d/%d): url=%s, name=%q, 测速地址=%s:%d, 携带旧Key=%t(fp=%s)",
			attempt, retries, registerURL, name, address, port, hasKey, keyFp)

		result, err := registerOnce(registerURL, token, name, address, protocol, port, metadataJSON, existingKey)
		if err == nil {
			statusWord := "新生成Key"
			if result.Reused {
				statusWord = "复用旧Key"
			}
			log.Printf("[node-agent] 注册请求成功 (第 %d 次尝试): node_id=%d, Key状态=%s, Key指纹=%s",
				attempt, result.ID, statusWord, security.KeyFingerprint(result.NodeKey))
			return result, nil
		}

		lastErr = err
		log.Printf("[node-agent] 注册尝试失败 (第 %d/%d 次): %v", attempt, retries, err)

		if attempt < retries {
			log.Printf("[node-agent] 将在 %v 后进行第 %d 次重试...", delay, attempt+1)
			time.Sleep(delay)
			delay *= 2
			if delay > registerMaxDelay {
				delay = registerMaxDelay
			}
		}
	}

	log.Printf("[node-agent] 自动注册最终失败: 已耗尽全部 %d 次重试，最后错误: %v", retries, lastErr)
	return nil, lastErr
}
