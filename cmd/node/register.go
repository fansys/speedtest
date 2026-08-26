package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
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

// registerOnce 发起一次自助注册请求，失败返回 error（不包含 REGISTRATION_TOKEN）。
func registerOnce(registerURL, token, name, address, protocol string, port int, metadataJSON, existingKey string) (*registerResult, error) {
	u, err := url.Parse(registerURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("NODE_REGISTER_URL 只允许 http/https 且必须是合法 URL")
	}

	payload := map[string]any{"name": name, "address": address, "port": port, "protocol": protocol}
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
		return nil, fmt.Errorf("注册请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("注册被服务端拒绝: HTTP %d", resp.StatusCode)
	}

	var out registerResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("注册响应不是合法 JSON: %w", err)
	}
	return &out, nil
}

// registerWithRetries 指数退避重试注册：1、2、4、8、16... 秒（封顶 registerMaxDelay）。
func registerWithRetries(registerURL, token, name, address, protocol string, port int, metadataJSON, existingKey string, retries int) (*registerResult, error) {
	if retries < 1 {
		retries = 1
	}
	var lastErr error
	delay := time.Second
	for attempt := 1; attempt <= retries; attempt++ {
		result, err := registerOnce(registerURL, token, name, address, protocol, port, metadataJSON, existingKey)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt < retries {
			time.Sleep(delay)
			delay *= 2
			if delay > registerMaxDelay {
				delay = registerMaxDelay
			}
		}
	}
	return nil, lastErr
}
