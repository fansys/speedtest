package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"librespeed-service/internal/config"
)

const testNodeKey = "test-node-key-0123456789abcdef"

func TestNodeAuth(t *testing.T) {
	srv, err := newNodeServer(testNodeKey, "test-node", 1024*1024)
	if err != nil {
		t.Fatalf("newNodeServer: %v", err)
	}
	handler := srv.routes()

	cases := []struct {
		name       string
		header     func(r *http.Request)
		wantStatus int
	}{
		{"missing key", func(r *http.Request) {}, http.StatusUnauthorized},
		{"wrong X-Node-Key", func(r *http.Request) { r.Header.Set("X-Node-Key", "wrong-key") }, http.StatusUnauthorized},
		{"wrong bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer wrong-key") }, http.StatusUnauthorized},
		{"correct X-Node-Key", func(r *http.Request) { r.Header.Set("X-Node-Key", testNodeKey) }, http.StatusOK},
		{"correct bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+testNodeKey) }, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			tc.header(req)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestNodeCORS(t *testing.T) {
	srv, err := newNodeServer(testNodeKey, "test-node", 1024*1024)
	if err != nil {
		t.Fatalf("newNodeServer: %v", err)
	}
	handler := srv.routes()

	// OPTIONS preflight
	req := httptest.NewRequest(http.MethodOptions, "/upload", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", rec.Code)
	}
	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want *", origin)
	}
	if methods := rec.Header().Get("Access-Control-Allow-Methods"); methods == "" {
		t.Fatal("expected Access-Control-Allow-Methods header")
	}
}

func TestNodePingAuthenticated(t *testing.T) {
	srv, err := newNodeServer(testNodeKey, "test-node", 1024*1024)
	if err != nil {
		t.Fatalf("newNodeServer: %v", err)
	}
	handler := srv.routes()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Node-Key", testNodeKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("ping status = %d, want 204", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated ping status = %d, want 401", rec.Code)
	}
}

func TestNodeDownloadUpload(t *testing.T) {
	srv, err := newNodeServer(testNodeKey, "test-node", 1024*1024)
	if err != nil {
		t.Fatalf("newNodeServer: %v", err)
	}
	handler := srv.routes()

	req := httptest.NewRequest(http.MethodGet, "/download?bytes=2048", nil)
	req.Header.Set("X-Node-Key", testNodeKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 2048 {
		t.Fatalf("download body len = %d, want 2048", rec.Body.Len())
	}

	uploadBody := bytes.NewReader(bytes.Repeat([]byte{'x'}, 4096))
	req = httptest.NewRequest(http.MethodPost, "/upload", uploadBody)
	req.Header.Set("X-Node-Key", testNodeKey)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
}

func TestNodeUploadLimitExceeded(t *testing.T) {
	maxBytes := int64(1024)
	srv, err := newNodeServer(testNodeKey, "test-node", maxBytes)
	if err != nil {
		t.Fatalf("newNodeServer: %v", err)
	}
	handler := srv.routes()

	// 上传 2048 字节，超过 1024 限制 -> 应返回 413
	uploadBody := bytes.NewReader(bytes.Repeat([]byte{'x'}, 2048))
	req := httptest.NewRequest(http.MethodPost, "/upload", uploadBody)
	req.Header.Set("X-Node-Key", testNodeKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("upload over limit status = %d, want 413, body=%s", rec.Code, rec.Body.String())
	}
}

func TestNodeDownloadClamped(t *testing.T) {
	maxBytes := int64(4096)
	srv, err := newNodeServer(testNodeKey, "test-node", maxBytes)
	if err != nil {
		t.Fatalf("newNodeServer: %v", err)
	}
	handler := srv.routes()

	// 请求 100000 字节，应截断到 maxBytes (4096)
	req := httptest.NewRequest(http.MethodGet, "/download?bytes=100000", nil)
	req.Header.Set("X-Node-Key", testNodeKey)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != int(maxBytes) {
		t.Fatalf("download body len = %d, want %d", rec.Body.Len(), maxBytes)
	}
}

func TestResolveNodeKeyAutoRegister(t *testing.T) {
	dir := t.TempDir()
	iniPath := filepath.Join(dir, "node.ini")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Registration-Token") != "test-reg-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)

		reused := false
		if payload["existing_node_key"] == testNodeKey {
			reused = true
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":       101,
			"node_key": testNodeKey,
			"reused":   reused,
		})
	}))
	defer mockServer.Close()

	cfg := &config.NodeConfig{
		RegisterURL:       mockServer.URL,
		RegistrationToken: "test-reg-token",
		Name:              "test-node",
		Address:           "127.0.0.1",
		Port:              8081,
		Protocol:          "http",
		NodeIni:           iniPath,
		RegisterRetries:   2,
	}

	key, err := resolveNodeKey(cfg)
	if err != nil {
		t.Fatalf("resolveNodeKey failed: %v", err)
	}
	if key != testNodeKey {
		t.Fatalf("got key %q, want %q", key, testNodeKey)
	}

	// 验证 node.ini 是否生成且有效
	state, err := config.LoadNodeState(iniPath)
	if err != nil || state == nil {
		t.Fatalf("LoadNodeState failed: %v", err)
	}
	if state["node_key"] != testNodeKey {
		t.Fatalf("state[node_key] = %q, want %q", state["node_key"], testNodeKey)
	}

	// 再次调用 resolveNodeKey，应当复用并更新
	key2, err := resolveNodeKey(cfg)
	if err != nil || key2 != testNodeKey {
		t.Fatalf("resolveNodeKey reuse failed: key=%q, err=%v", key2, err)
	}

	// 显式 NODE_KEY 时应当直接返回并不读写 node.ini
	os.Remove(iniPath)
	cfgExplicit := &config.NodeConfig{
		NodeKey: "explicit-key-1234567890123456",
		NodeIni: iniPath,
	}
	explicitKey, err := resolveNodeKey(cfgExplicit)
	if err != nil || explicitKey != "explicit-key-1234567890123456" {
		t.Fatalf("explicit resolve failed: %v", err)
	}
}
