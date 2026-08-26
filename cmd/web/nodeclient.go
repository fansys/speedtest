package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"librespeed-service/internal/netguard"
	"librespeed-service/internal/store"
)

// nodeClientError 表示节点不可达、拒绝了 node_key，或返回了非预期响应。
type nodeClientError struct{ msg string }

func (e *nodeClientError) Error() string { return e.msg }

func nodeErr(format string, args ...any) error {
	return &nodeClientError{msg: fmt.Sprintf(format, args...)}
}

func durationFromSeconds(sec float64) time.Duration {
	return time.Duration(sec * float64(time.Second))
}

// nodeBaseURL 每次调用前都重新校验节点地址，缩小 DNS rebinding 的窗口。
func (s *server) nodeBaseURL(node *store.Node) (string, error) {
	resolved, err := netguard.ValidateNodeTarget(node.Address, node.Port, node.Protocol, netguard.ValidateNodeTargetOptions{
		AllowPrivate:     s.settings.AllowPrivateNodes,
		AllowedProtocols: s.settings.AllowedNodeProtocols,
		Resolve:          true,
	})
	if err != nil {
		return "", nodeErr("节点地址不再被允许访问: %s", err.Error())
	}
	return resolved.BaseURL(), nil
}

func (s *server) checkHealth(node *store.Node, nodeKey string) error {
	base, err := s.nodeBaseURL(node)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeHealthTimeout)}
	req, err := http.NewRequest(http.MethodGet, base+"/healthz", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Node-Key", nodeKey)
	resp, err := client.Do(req)
	if err != nil {
		return nodeErr("无法连接节点: %s", err.Error())
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return nodeErr("节点拒绝了 node_key")
	}
	if resp.StatusCode != http.StatusOK {
		return nodeErr("节点返回异常状态码 %d", resp.StatusCode)
	}
	return nil
}

func (s *server) measurePing(node *store.Node, nodeKey string, count int) (pingResult, error) {
	base, err := s.nodeBaseURL(node)
	if err != nil {
		return pingResult{}, err
	}
	if count <= 0 {
		count = s.settings.PingCount
	}
	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeConnectTimeout)}

	latencies := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		start := time.Now()
		req, err := http.NewRequest(http.MethodGet, base+"/ping", nil)
		if err != nil {
			return pingResult{}, err
		}
		req.Header.Set("X-Node-Key", nodeKey)
		resp, err := client.Do(req)
		if err != nil {
			return pingResult{}, nodeErr("ping 失败: %s", err.Error())
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			return pingResult{}, nodeErr("节点拒绝了 node_key")
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return pingResult{}, nodeErr("节点返回异常状态码 %d", resp.StatusCode)
		}
		latencies = append(latencies, time.Since(start).Seconds()*1000)
	}

	minV, maxV, sum := latencies[0], latencies[0], 0.0
	for _, v := range latencies {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
		sum += v
	}
	return pingResult{
		Count:    len(latencies),
		MinMs:    minV,
		AvgMs:    sum / float64(len(latencies)),
		MaxMs:    maxV,
		JitterMs: maxV - minV,
	}, nil
}

func (s *server) measureDownload(node *store.Node, nodeKey string, numBytes int64) (transferResult, error) {
	base, err := s.nodeBaseURL(node)
	if err != nil {
		return transferResult{}, err
	}
	if numBytes <= 0 {
		numBytes = s.settings.DefaultDownloadBytes
	}
	if numBytes > s.settings.MaxTestBytes {
		numBytes = s.settings.MaxTestBytes
	}

	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeRequestTimeout)}
	u := base + "/download?" + url.Values{"bytes": {strconv.FormatInt(numBytes, 10)}}.Encode()
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return transferResult{}, err
	}
	req.Header.Set("X-Node-Key", nodeKey)

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return transferResult{}, nodeErr("下载测速失败: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return transferResult{}, nodeErr("节点拒绝了 node_key")
	}
	if resp.StatusCode != http.StatusOK {
		return transferResult{}, nodeErr("节点返回异常状态码 %d", resp.StatusCode)
	}
	total, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return transferResult{}, nodeErr("下载测速失败: %s", err.Error())
	}
	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	mbps := (float64(total) * 8 / 1_000_000) / elapsed
	return transferResult{Bytes: total, DurationMs: elapsed * 1000, Mbps: mbps}, nil
}

func (s *server) measureUpload(node *store.Node, nodeKey string, numBytes int64) (transferResult, error) {
	base, err := s.nodeBaseURL(node)
	if err != nil {
		return transferResult{}, err
	}
	if numBytes <= 0 {
		numBytes = s.settings.DefaultUploadBytes
	}
	if numBytes > s.settings.MaxTestBytes {
		numBytes = s.settings.MaxTestBytes
	}

	chunkSize := s.settings.StreamChunkBytes
	if int64(chunkSize) > numBytes {
		chunkSize = int(numBytes)
	}
	chunk := make([]byte, chunkSize)
	if chunkSize > 0 {
		if _, err := rand.Read(chunk); err != nil {
			return transferResult{}, err
		}
	}

	client := &http.Client{Timeout: durationFromSeconds(s.settings.NodeRequestTimeout)}
	req, err := http.NewRequest(http.MethodPost, base+"/upload", &repeatReader{chunk: chunk, remaining: numBytes})
	if err != nil {
		return transferResult{}, err
	}
	req.Header.Set("X-Node-Key", nodeKey)
	req.ContentLength = numBytes

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return transferResult{}, nodeErr("上传测速失败: %s", err.Error())
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		return transferResult{}, nodeErr("节点拒绝了 node_key")
	}
	if resp.StatusCode != http.StatusOK {
		return transferResult{}, nodeErr("节点返回异常状态码 %d", resp.StatusCode)
	}
	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	mbps := (float64(numBytes) * 8 / 1_000_000) / elapsed
	return transferResult{Bytes: numBytes, DurationMs: elapsed * 1000, Mbps: mbps}, nil
}

// repeatReader 循环输出 chunk 的内容，总长度截止到 remaining；用于生成上传测速的
// 请求体，避免为大体积测试一次性分配整块内存。
type repeatReader struct {
	chunk     []byte
	remaining int64
	pos       int
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.remaining <= 0 || len(r.chunk) == 0 {
		return 0, io.EOF
	}
	max := len(p)
	if int64(max) > r.remaining {
		max = int(r.remaining)
	}
	total := 0
	for total < max {
		n := copy(p[total:max], r.chunk[r.pos:])
		total += n
		r.pos = (r.pos + n) % len(r.chunk)
	}
	r.remaining -= int64(total)
	return total, nil
}
