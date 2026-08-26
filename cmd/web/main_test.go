package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"librespeed-service/internal/config"
	"librespeed-service/internal/store"
)

func testSettings() *config.Settings {
	return &config.Settings{
		AdminToken:             "admin-token-1234567890",
		RegistrationToken:      "registration-token-1234567890",
		SecretKey:              "secret-key-12345678901234567890",
		AllowPrivateNodes:      true,
		AllowedNodeProtocols:   []string{"http", "https"},
		NodeConnectTimeout:     5,
		NodeRequestTimeout:     30,
		NodeHealthTimeout:      5,
		MaxTestBytes:           512 * 1024 * 1024,
		DefaultDownloadBytes:   1024,
		DefaultUploadBytes:     1024,
		StreamChunkBytes:       1024,
		PingCount:              3,
		EnableCentralSpeedtest: true,
		StoreNodeKeySealed:     true,
	}
}

func newTestServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	staticDir := t.TempDir()
	os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<!DOCTYPE html><html><body>LibreSpeed Test</body></html>"), 0o644)
	os.WriteFile(filepath.Join(staticDir, "style.css"), []byte("body { background: #000; }"), 0o644)
	os.WriteFile(filepath.Join(staticDir, "app.js"), []byte("console.log('test');"), 0o644)

	return NewServer(st, testSettings(), staticDir), staticDir
}

func TestHealthz(t *testing.T) {
	handler, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v, want ok", body["status"])
	}
	if body["enable_central_speedtest"] != true {
		t.Fatalf("enable_central_speedtest = %v, want true", body["enable_central_speedtest"])
	}
}

func TestConfigEndpoint(t *testing.T) {
	handler, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body["enable_central_speedtest"] != true {
		t.Fatalf("expected enable_central_speedtest true, got %v", body["enable_central_speedtest"])
	}
}

func TestUIIndexAndStatic(t *testing.T) {
	handler, _ := newTestServer(t)

	// GET /
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "LibreSpeed Test") {
		t.Fatalf("GET / unexpected content: %s", rec.Body.String())
	}

	// GET /static/style.css
	req = httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/style.css status = %d, want 200", rec.Code)
	}

	// GET /static/app.js
	req = httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/app.js status = %d, want 200", rec.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler, _ := newTestServer(t)
	settings := testSettings()

	// 1. Admin Auth
	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing admin token, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid Bearer token, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid X-Admin-Token, got %d", rec.Code)
	}

	// 2. Registration Auth: 支持 Admin Token 或 Registration Token
	req = httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing token, got %d", rec.Code)
	}

	// 使用 Admin Token 直接注册
	req = httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{}`))
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatal("valid Admin Token was rejected on /api/register")
	}

	// 使用 Registration Token 注册
	req = httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+settings.RegistrationToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Fatal("valid Bearer registration token was rejected")
	}
}

func doRegisterAuto(t *testing.T, handler http.Handler, settings *config.Settings, payload map[string]any) autoRegisterOut {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/register/auto", bytes.NewReader(body))
	req.Header.Set("X-Registration-Token", settings.RegistrationToken)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("register/auto status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var out autoRegisterOut
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	return out
}

func TestRegisterAutoGenerateAndReuse(t *testing.T) {
	handler, _ := newTestServer(t)
	settings := testSettings()

	first := doRegisterAuto(t, handler, settings, map[string]any{
		"name": "node-1", "address": "127.0.0.1", "port": 8081, "protocol": "http",
	})
	if first.NodeKey == "" {
		t.Fatal("expected non-empty node_key")
	}
	if first.Reused {
		t.Fatal("first registration should not be reused")
	}

	second := doRegisterAuto(t, handler, settings, map[string]any{
		"name": "node-1", "address": "127.0.0.1", "port": 8081, "protocol": "http",
		"existing_node_key": first.NodeKey,
	})
	if !second.Reused {
		t.Fatal("second registration with matching existing_node_key should be reused")
	}
	if second.NodeKey != first.NodeKey {
		t.Fatalf("reused key mismatch: got %q want %q", second.NodeKey, first.NodeKey)
	}

	// GET /api/nodes 不应该泄露 node_key 明文
	req := httptest.NewRequest(http.MethodGet, "/api/nodes", nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list nodes status = %d", rec.Code)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(first.NodeKey)) {
		t.Fatal("node list response leaked node_key")
	}
}

func TestNodeCRUD(t *testing.T) {
	handler, _ := newTestServer(t)
	settings := testSettings()

	reg := doRegisterAuto(t, handler, settings, map[string]any{
		"name": "node-crud", "address": "127.0.0.1", "port": 8082, "protocol": "http",
	})
	nodeID := reg.ID

	// 1. Get Node
	req := httptest.NewRequest(http.MethodGet, "/api/nodes/"+strconv.FormatInt(nodeID, 10), nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get node status = %d", rec.Code)
	}

	// 2. Disable Node
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/disable", nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable node status = %d", rec.Code)
	}
	var out nodeOut
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Enabled {
		t.Fatal("expected node to be disabled")
	}

	// 3. Enable Node
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/enable", nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("enable node status = %d", rec.Code)
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if !out.Enabled {
		t.Fatal("expected node to be enabled")
	}

	// 4. Delete Node
	req = httptest.NewRequest(http.MethodDelete, "/api/nodes/"+strconv.FormatInt(nodeID, 10), nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete node status = %d, want 204", rec.Code)
	}

	// 5. Get after delete -> 404
	req = httptest.NewRequest(http.MethodGet, "/api/nodes/"+strconv.FormatInt(nodeID, 10), nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get deleted node status = %d, want 404", rec.Code)
	}
}

func TestNodeProxyAndSpeedtestEndToEnd(t *testing.T) {
	var mockNodeKey string
	mockNode := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Node-Key") != mockNodeKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		case "/ping":
			w.WriteHeader(http.StatusNoContent)
		case "/download":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(bytes.Repeat([]byte{0x42}, 1024))
		case "/upload":
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"bytes":1024,"duration_ms":10.0,"mbps":100.0}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockNode.Close()

	nodeHostPort := strings.TrimPrefix(mockNode.URL, "http://")
	host, portStr, _ := strings.Cut(nodeHostPort, ":")
	port, _ := strconv.Atoi(portStr)

	handler, _ := newTestServer(t)
	settings := testSettings()

	reg := doRegisterAuto(t, handler, settings, map[string]any{
		"name": "mock-node", "address": host, "port": port, "protocol": "http",
	})
	mockNodeKey = reg.NodeKey
	nodeID := reg.ID

	// 1. POST /api/nodes/{id}/health
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/health", nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health check status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var hr healthResult
	json.Unmarshal(rec.Body.Bytes(), &hr)
	if hr.Status != "online" {
		t.Fatalf("health status = %q, want online", hr.Status)
	}

	// 2. POST /api/nodes/{id}/speedtest
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/speedtest", nil)
	req.Header.Set("X-Admin-Token", settings.AdminToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("speedtest status = %d, body=%s", rec.Code, rec.Body.String())
	}
	var sr speedtestResult
	json.Unmarshal(rec.Body.Bytes(), &sr)
	if sr.Ping == nil || sr.Download == nil || sr.Upload == nil {
		t.Fatalf("speedtest result missing components: %+v", sr)
	}

	// 3. GET /api/nodes/{id}/ping
	req = httptest.NewRequest(http.MethodGet, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/ping", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy ping status = %d", rec.Code)
	}

	// 4. GET /api/nodes/{id}/download
	req = httptest.NewRequest(http.MethodGet, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/download?bytes=1024", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy download status = %d", rec.Code)
	}
	if rec.Body.Len() != 1024 {
		t.Fatalf("proxy download len = %d, want 1024", rec.Body.Len())
	}

	// 5. POST /api/nodes/{id}/upload
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.FormatInt(nodeID, 10)+"/upload", strings.NewReader("hello-upload"))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("proxy upload status = %d", rec.Code)
	}
}

func TestDirectSpeedtestEndpoints(t *testing.T) {
	handler, _ := newTestServer(t)

	// 1. GET /api/speedtest/ping
	req := httptest.NewRequest(http.MethodGet, "/api/speedtest/ping", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("direct ping status = %d, want 204", rec.Code)
	}

	// 2. GET /api/speedtest/download
	req = httptest.NewRequest(http.MethodGet, "/api/speedtest/download?bytes=2048", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("direct download status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() != 2048 {
		t.Fatalf("direct download body len = %d, want 2048", rec.Body.Len())
	}

	// 3. POST /api/speedtest/upload
	req = httptest.NewRequest(http.MethodPost, "/api/speedtest/upload", bytes.NewReader(bytes.Repeat([]byte{'z'}, 4096)))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("direct upload status = %d, want 200", rec.Code)
	}
}

func TestCentralSpeedtestDisabled(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	settings := testSettings()
	settings.EnableCentralSpeedtest = false
	handler := NewServer(st, settings, t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/api/speedtest/ping", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when central speedtest disabled, got %d", rec.Code)
	}
}

func TestSecurityHeadersAndCORS(t *testing.T) {
	handler, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodOptions, "/api/nodes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("missing Access-Control-Allow-Origin")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing X-Content-Type-Options: nosniff")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing X-Frame-Options: DENY")
	}
}
