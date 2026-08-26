package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"librespeed-service/internal/security"
)

type nodeStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *nodeStatusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

// nodeServer 是 LibreSpeed 风格的独立节点 agent：/healthz /ping /download /upload。
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

	return s.corsAndLoggingMiddleware(mux)
}

func (s *nodeServer) corsAndLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &nodeStatusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Node-Key, Authorization, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Method == http.MethodOptions {
			rec.WriteHeader(http.StatusNoContent)
			log.Printf("[node-agent] HTTP OPTIONS %s 204 %v client=%s", r.URL.Path, time.Since(start), r.RemoteAddr)
			return
		}

		next.ServeHTTP(rec, r)

		if r.URL.Path != "/healthz" {
			log.Printf("[node-agent] HTTP %s %s %d %v client=%s", r.Method, r.URL.Path, rec.statusCode, time.Since(start), r.RemoteAddr)
		}
	})
}

func (s *nodeServer) authenticate(r *http.Request) bool {
	supplied := security.PickToken(r.Header.Get("X-Node-Key"), r.Header.Get("Authorization"))
	return supplied != "" && security.ConstantTimeEquals(supplied, s.nodeKey)
}

func (s *nodeServer) unauthorized(w http.ResponseWriter, r *http.Request) {
	log.Printf("[node-agent] 拒绝未授权请求: %s %s 来源=%s (node_key 无效或缺失)", r.Method, r.URL.Path, r.RemoteAddr)
	writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "node_key 无效或缺失"})
}

func (s *nodeServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "name": s.name})
}

func (s *nodeServer) handlePing(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusNoContent)
}

func (s *nodeServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		s.unauthorized(w, r)
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

	log.Printf("[node-agent] 收到下载测速请求: bytes=%d, client=%s", total, r.RemoteAddr)

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
		s.unauthorized(w, r)
		return
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	start := time.Now()
	total, err := io.Copy(io.Discard, io.LimitReader(r.Body, s.maxTestBytes+1))
	if err != nil {
		log.Printf("[node-agent] 上传测速读取失败: %v, client=%s", err, r.RemoteAddr)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "读取上传数据失败"})
		return
	}
	if total > s.maxTestBytes {
		log.Printf("[node-agent] 上传测速超过大小上限 (%d > %d), client=%s", total, s.maxTestBytes, r.RemoteAddr)
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"detail": "上传数据超过节点允许的上限"})
		return
	}

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	mbps := (float64(total) * 8 / 1_000_000) / elapsed

	log.Printf("[node-agent] 上传测速完成: bytes=%d, mbps=%.2f, duration=%.1fms, client=%s",
		total, mbps, elapsed*1000, r.RemoteAddr)

	writeJSON(w, http.StatusOK, map[string]any{
		"bytes":       total,
		"duration_ms": elapsed * 1000,
		"mbps":        mbps,
	})
}
