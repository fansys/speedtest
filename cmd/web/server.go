package main

import (
	"crypto/rand"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"librespeed-service/internal/config"
	"librespeed-service/internal/security"
	"librespeed-service/internal/store"
)

type webStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rec *webStatusRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

type server struct {
	store         *store.Store
	settings      *config.Settings
	staticDir     string
	downloadChunk []byte
}

// NewServer 构造中心 Web 服务的 http.Handler。staticDir 是 static/ 目录的路径。
func NewServer(st *store.Store, settings *config.Settings, staticDir string) http.Handler {
	chunk := make([]byte, settings.StreamChunkBytes)
	if len(chunk) == 0 {
		chunk = make([]byte, 64*1024)
	}
	_, _ = rand.Read(chunk)

	s := &server{
		store:         st,
		settings:      settings,
		staticDir:     staticDir,
		downloadChunk: chunk,
	}

	mux := http.NewServeMux()

	// 基础与静态资源
	mux.HandleFunc("GET /healthz", s.handleServiceHealthz)
	mux.HandleFunc("GET /api/config", s.handleConfig)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// 注册相关（支持 Admin Token 或 Registration Token）
	mux.Handle("POST /api/register", s.requireRegistrationOrAdmin(http.HandlerFunc(s.handleRegister)))
	mux.Handle("POST /api/register/auto", s.requireRegistrationOrAdmin(http.HandlerFunc(s.handleRegisterAuto)))

	// 节点管理 API（需要 Admin Token）
	mux.Handle("GET /api/nodes", s.requireAdmin(http.HandlerFunc(s.handleListNodes)))
	mux.Handle("GET /api/nodes/{id}", s.requireAdmin(http.HandlerFunc(s.handleGetNode)))
	mux.Handle("POST /api/nodes/{id}/enable", s.requireAdmin(http.HandlerFunc(s.handleEnableNode)))
	mux.Handle("POST /api/nodes/{id}/disable", s.requireAdmin(http.HandlerFunc(s.handleDisableNode)))
	mux.Handle("DELETE /api/nodes/{id}", s.requireAdmin(http.HandlerFunc(s.handleDeleteNode)))
	mux.Handle("POST /api/nodes/{id}/health", s.requireAdmin(http.HandlerFunc(s.handleHealthCheck)))
	mux.Handle("POST /api/nodes/{id}/speedtest", s.requireAdmin(http.HandlerFunc(s.handleSpeedtest)))

	// 节点代理测速 API（供浏览器实时流式测速）
	mux.HandleFunc("GET /api/nodes/{id}/ping", s.handleNodePingProxy)
	mux.HandleFunc("GET /api/nodes/{id}/download", s.handleNodeDownloadProxy)
	mux.HandleFunc("POST /api/nodes/{id}/upload", s.handleNodeUploadProxy)

	// Web 中心自测 API（受 EnableCentralSpeedtest 环境变量控制）
	mux.HandleFunc("GET /api/speedtest/ping", s.handleDirectPing)
	mux.HandleFunc("GET /api/speedtest/download", s.handleDirectDownload)
	mux.HandleFunc("POST /api/speedtest/upload", s.handleDirectUpload)

	return s.corsAndLoggingMiddleware(mux)
}

func (s *server) corsAndLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &webStatusRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, X-Registration-Token, X-Node-Key, Authorization, Cache-Control, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		if r.Method == http.MethodOptions {
			rec.WriteHeader(http.StatusNoContent)
			log.Printf("[web] HTTP OPTIONS %s 204 %v client=%s", r.URL.Path, time.Since(start), r.RemoteAddr)
			return
		}

		next.ServeHTTP(rec, r)

		if r.URL.Path != "/healthz" {
			log.Printf("[web] HTTP %s %s %d %v client=%s", r.Method, r.URL.Path, rec.statusCode, time.Since(start), r.RemoteAddr)
		}
	})
}

func (s *server) handleServiceHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                   "ok",
		"enable_central_speedtest": s.settings.EnableCentralSpeedtest,
	})
}

func (s *server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]any{
		"status":                   "ok",
		"enable_central_speedtest": s.settings.EnableCentralSpeedtest,
		"ping_count":               s.settings.PingCount,
	})
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
}

func (s *server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := security.PickToken(r.Header.Get("X-Admin-Token"), r.Header.Get("Authorization"))
		if token == "" || !security.ConstantTimeEquals(token, s.settings.AdminToken) {
			log.Printf("[web] 管理接口未授权拒绝: %s %s, 来源IP=%s (ADMIN_TOKEN 无效或缺失)", r.Method, r.URL.Path, r.RemoteAddr)
			writeError(w, http.StatusUnauthorized, "管理令牌无效或缺失")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireRegistrationOrAdmin 鉴权：接受正确的 ADMIN_TOKEN 或 REGISTRATION_TOKEN（网页管理端只需输入 Admin Token 即可完成全部操作）。
func (s *server) requireRegistrationOrAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminToken := security.PickToken(r.Header.Get("X-Admin-Token"), r.Header.Get("Authorization"))
		if adminToken != "" && security.ConstantTimeEquals(adminToken, s.settings.AdminToken) {
			next.ServeHTTP(w, r)
			return
		}

		regToken := security.PickToken(r.Header.Get("X-Registration-Token"), r.Header.Get("Authorization"))
		if regToken != "" && security.ConstantTimeEquals(regToken, s.settings.RegistrationToken) {
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("[web] 注册接口未授权拒绝: %s %s, 来源IP=%s (ADMIN_TOKEN 或 REGISTRATION_TOKEN 无效或缺失)", r.Method, r.URL.Path, r.RemoteAddr)
		writeError(w, http.StatusUnauthorized, "令牌无效或缺失")
	})
}

func (s *server) handleDirectPing(w http.ResponseWriter, r *http.Request) {
	if !s.settings.EnableCentralSpeedtest {
		writeError(w, http.StatusForbidden, "中心节点测速功能已禁用")
		return
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleDirectDownload(w http.ResponseWriter, r *http.Request) {
	if !s.settings.EnableCentralSpeedtest {
		writeError(w, http.StatusForbidden, "中心节点测速功能已禁用")
		return
	}

	total := s.settings.DefaultDownloadBytes
	if v := r.URL.Query().Get("bytes"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			total = n
		}
	}
	if total > s.settings.MaxTestBytes {
		total = s.settings.MaxTestBytes
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

func (s *server) handleDirectUpload(w http.ResponseWriter, r *http.Request) {
	if !s.settings.EnableCentralSpeedtest {
		writeError(w, http.StatusForbidden, "中心节点测速功能已禁用")
		return
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	total, err := io.Copy(io.Discard, io.LimitReader(r.Body, s.settings.MaxTestBytes+1))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取上传数据失败")
		return
	}
	if total > s.settings.MaxTestBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "上传数据超过允许的最大限制")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"bytes": total,
	})
}
