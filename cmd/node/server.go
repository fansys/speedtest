package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"librespeed-service/internal/security"
)

// nodeServer 是 LibreSpeed 风格的独立节点 agent：/healthz /ping /download /upload，
// 全部要求请求头携带正确的 node_key（X-Node-Key 或 Authorization: Bearer）。
type nodeServer struct {
	nodeKey       string
	name          string
	maxTestBytes  int64
	downloadChunk []byte
}

func newNodeServer(nodeKey, name string, maxTestBytes int64) (*nodeServer, error) {
	if len(nodeKey) < security.MinNodeKeyLength {
		return nil, fmt.Errorf("node_key 长度至少 %d 个字符", security.MinNodeKeyLength)
	}
	chunk := make([]byte, 64*1024)
	if _, err := rand.Read(chunk); err != nil {
		return nil, err
	}
	return &nodeServer{nodeKey: nodeKey, name: name, maxTestBytes: maxTestBytes, downloadChunk: chunk}, nil
}

func (s *nodeServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /ping", s.handlePing)
	mux.HandleFunc("GET /download", s.handleDownload)
	mux.HandleFunc("POST /upload", s.handleUpload)

	return s.corsAndSecurityMiddleware(mux)
}

func (s *nodeServer) corsAndSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Node-Key, Authorization, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *nodeServer) authenticate(r *http.Request) bool {
	supplied := security.PickToken(r.Header.Get("X-Node-Key"), r.Header.Get("Authorization"))
	return supplied != "" && security.ConstantTimeEquals(supplied, s.nodeKey)
}

func (s *nodeServer) unauthorized(w http.ResponseWriter) {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "node_key 无效或缺失"})
}

func (s *nodeServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w)
		return
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "name": s.name})
}

func (s *nodeServer) handlePing(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w)
		return
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusNoContent)
}

func (s *nodeServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w)
		return
	}

	total := int64(10_000_000)
	if v := r.URL.Query().Get("bytes"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			total = n
		}
	}
	if total > s.maxTestBytes {
		total = s.maxTestBytes
	}
	if total < 1 {
		total = 1
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(total, 10))
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusOK)

	chunk := s.downloadChunk
	sent := int64(0)
	for sent < total {
		n := int64(len(chunk))
		if remain := total - sent; remain < n {
			n = remain
		}
		if _, err := w.Write(chunk[:n]); err != nil {
			return
		}
		sent += n
	}
}

func (s *nodeServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w)
		return
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	start := time.Now()
	total, err := io.Copy(io.Discard, io.LimitReader(r.Body, s.maxTestBytes+1))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "读取上传数据失败"})
		return
	}
	if total > s.maxTestBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"detail": "上传数据超过节点允许的上限"})
		return
	}

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	mbps := (float64(total) * 8 / 1_000_000) / elapsed
	writeJSON(w, http.StatusOK, map[string]any{
		"bytes":       total,
		"duration_ms": elapsed * 1000,
		"mbps":        mbps,
	})
}
