package main

import (
	"encoding/json"
	"time"

	"librespeed-service/internal/store"
)

// nodeOut 是节点的对外表示：绝不包含 node_key 明文或封存密文。
type nodeOut struct {
	ID             int64          `json:"id"`
	Name           string         `json:"name"`
	Address        string         `json:"address"`
	Port           int            `json:"port"`
	Protocol       string         `json:"protocol"`
	KeyFingerprint string         `json:"key_fingerprint"`
	Enabled        bool           `json:"enabled"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	LastSeenAt     *time.Time     `json:"last_seen_at"`
	LastStatus     string         `json:"last_status"`
	LastLatencyMs  *float64       `json:"last_latency_ms"`
	LastError      *string        `json:"last_error"`
}

func nodeToOut(n *store.Node) (nodeOut, error) {
	var metadata map[string]any
	if n.MetadataJSON != nil && *n.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(*n.MetadataJSON), &metadata); err != nil {
			return nodeOut{}, err
		}
	}
	return nodeOut{
		ID:             n.ID,
		Name:           n.Name,
		Address:        n.Address,
		Port:           n.Port,
		Protocol:       n.Protocol,
		KeyFingerprint: n.KeyFingerprint,
		Enabled:        n.Enabled,
		Metadata:       metadata,
		CreatedAt:      n.CreatedAt,
		UpdatedAt:      n.UpdatedAt,
		LastSeenAt:     n.LastSeenAt,
		LastStatus:     n.LastStatus,
		LastLatencyMs:  n.LastLatencyMs,
		LastError:      n.LastError,
	}, nil
}

// registerRequest 是 POST /api/register 与 /api/register/auto 共用的请求体；
// existing_node_key 只在自动注册路径下生效。
type registerRequest struct {
	Name            string         `json:"name"`
	Address         string         `json:"address"`
	Port            int            `json:"port"`
	Protocol        string         `json:"protocol"`
	Metadata        map[string]any `json:"metadata"`
	ExistingNodeKey *string        `json:"existing_node_key"`
}

// registerOut 是注册成功的响应；node_key 是一次性凭据，只在这个响应里出现一次。
type registerOut struct {
	nodeOut
	NodeKey string `json:"node_key"`
}

type autoRegisterOut struct {
	registerOut
	Reused bool `json:"reused"`
}

// speedtestRequest 是 health / speedtest 的可选请求体。
type speedtestRequest struct {
	NodeKey       *string `json:"node_key"`
	PingCount     *int    `json:"ping_count"`
	DownloadBytes *int64  `json:"download_bytes"`
	UploadBytes   *int64  `json:"upload_bytes"`
}

type pingResult struct {
	Count    int     `json:"count"`
	MinMs    float64 `json:"min_ms"`
	AvgMs    float64 `json:"avg_ms"`
	MaxMs    float64 `json:"max_ms"`
	JitterMs float64 `json:"jitter_ms"`
}

type transferResult struct {
	Bytes      int64   `json:"bytes"`
	DurationMs float64 `json:"duration_ms"`
	Mbps       float64 `json:"mbps"`
}

type speedtestResult struct {
	NodeID   int64           `json:"node_id"`
	Ping     *pingResult     `json:"ping"`
	Download *transferResult `json:"download"`
	Upload   *transferResult `json:"upload"`
	Error    *string         `json:"error"`
}

type healthResult struct {
	NodeID    int64    `json:"node_id"`
	Status    string   `json:"status"`
	LatencyMs *float64 `json:"latency_ms"`
	Error     *string  `json:"error"`
}
