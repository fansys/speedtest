package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveNodeState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.ini")

	fields := map[string]string{
		"node_id":  "42",
		"node_key": "test-key-0123456789abcdef",
		"name":     "node-shanghai",
		"address":  "127.0.0.1",
		"port":     "8081",
		"protocol": "http",
	}

	if err := SaveNodeState(path, fields); err != nil {
		t.Fatalf("SaveNodeState failed: %v", err)
	}

	// 检查文件权限为 0600
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected permissions 0600, got %#o", perm)
	}

	loaded, err := LoadNodeState(path)
	if err != nil {
		t.Fatalf("LoadNodeState failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded state, got nil")
	}

	for k, v := range fields {
		if loaded[k] != v {
			t.Fatalf("field %q: got %q, want %q", k, loaded[k], v)
		}
	}
}

func TestLoadNodeStateNonExistent(t *testing.T) {
	loaded, err := LoadNodeState(filepath.Join(t.TempDir(), "nonexistent.ini"))
	if err != nil {
		t.Fatalf("unexpected error for nonexistent file: %v", err)
	}
	if loaded != nil {
		t.Fatal("expected nil for nonexistent file")
	}
}
