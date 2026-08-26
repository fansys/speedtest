package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func mockKeyFuncs() (func() string, func(string) string, func(string) string, func(string) (*string, error)) {
	gen := func() string { return "test-node-key-1234567890abcdef" }
	hash := func(k string) string { return "hash-" + k }
	fp := func(k string) string { return "fp-" + k[:4] }
	seal := func(k string) (*string, error) {
		s := "sealed-" + k
		return &s, nil
	}
	return gen, hash, fp, seal
}

func TestStoreCRUD(t *testing.T) {
	st, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	gen, hash, fp, seal := mockKeyFuncs()

	// 1. 注册新节点
	node, key, reused, err := st.RegisterOrReuse(RegisterParams{
		Name:        "node-beijing",
		Address:     "192.168.1.100",
		Port:        8081,
		Protocol:    "http",
		GenerateKey: gen,
		HashKey:     hash,
		Fingerprint: fp,
		SealKey:     seal,
	})
	if err != nil {
		t.Fatalf("RegisterOrReuse failed: %v", err)
	}
	if reused {
		t.Fatal("expected reused=false for new node")
	}
	if node.ID == 0 {
		t.Fatal("expected valid node ID")
	}
	if key != gen() {
		t.Fatalf("expected key %q, got %q", gen(), key)
	}

	// 2. Get 节点
	fetched, err := st.Get(node.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if fetched.Name != "node-beijing" {
		t.Fatalf("Name mismatch: %q", fetched.Name)
	}

	// 3. List 节点
	list, err := st.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 node, got %d", len(list))
	}

	// 4. UpdateHealthResult
	lat := 12.5
	errMsg := "connection reset"
	if err := st.UpdateHealthResult(node.ID, "error", &lat, &errMsg, false); err != nil {
		t.Fatalf("UpdateHealthResult failed: %v", err)
	}
	updated, _ := st.Get(node.ID)
	if updated.LastStatus != "error" || updated.LastLatencyMs == nil || *updated.LastLatencyMs != 12.5 {
		t.Fatalf("health status mismatch: %+v", updated)
	}

	// 5. SetEnabled
	disabled, err := st.SetEnabled(node.ID, false)
	if err != nil || disabled.Enabled {
		t.Fatalf("SetEnabled(false) failed: %+v, err=%v", disabled, err)
	}

	// 6. Delete
	if err := st.Delete(node.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	_, err = st.Get(node.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestStoreRegisterReuse(t *testing.T) {
	st, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	gen, hash, fp, seal := mockKeyFuncs()

	// 首次注册
	node1, key1, reused1, err := st.RegisterOrReuse(RegisterParams{
		Name:        "node-shanghai",
		Address:     "10.0.0.1",
		Port:        8081,
		Protocol:    "http",
		GenerateKey: gen,
		HashKey:     hash,
		Fingerprint: fp,
		SealKey:     seal,
	})
	if err != nil || reused1 {
		t.Fatalf("first reg failed: reused=%v, err=%v", reused1, err)
	}

	// 携带正确 existing_node_key 再次注册同一节点 -> 应该复用
	node2, key2, reused2, err := st.RegisterOrReuse(RegisterParams{
		Name:            "node-shanghai-renamed",
		Address:         "10.0.0.1",
		Port:            8081,
		Protocol:        "http",
		ExistingNodeKey: &key1,
		GenerateKey:     gen,
		HashKey:         hash,
		Fingerprint:     fp,
		SealKey:         seal,
	})
	if err != nil || !reused2 {
		t.Fatalf("second reg reuse failed: reused=%v, err=%v", reused2, err)
	}
	if node2.ID != node1.ID || key2 != key1 {
		t.Fatalf("node ID or key changed on reuse: id1=%d, id2=%d", node1.ID, node2.ID)
	}

	// 携带错误 key 注册同一节点 -> 重新生成 key，不再复用
	wrongKey := "wrong-key-value"
	node3, key3, reused3, err := st.RegisterOrReuse(RegisterParams{
		Name:            "node-shanghai",
		Address:         "10.0.0.1",
		Port:            8081,
		Protocol:        "http",
		ExistingNodeKey: &wrongKey,
		GenerateKey:     func() string { return "new-gen-key-999" },
		HashKey:         hash,
		Fingerprint:     fp,
		SealKey:         seal,
	})
	if err != nil || reused3 {
		t.Fatalf("third reg with wrong key: reused=%v, err=%v", reused3, err)
	}
	if key3 == key1 {
		t.Fatal("key should have rotated on mismatch")
	}
	if node3.ID != node1.ID {
		t.Fatalf("node ID changed on key rotation: id1=%d, id3=%d", node1.ID, node3.ID)
	}
}

func TestStoreFilePersistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nodes.db")

	st, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	gen, hash, fp, seal := mockKeyFuncs()
	node, _, _, err := st.RegisterOrReuse(RegisterParams{
		Name:        "persisted-node",
		Address:     "192.168.1.50",
		Port:        8081,
		Protocol:    "http",
		GenerateKey: gen,
		HashKey:     hash,
		Fingerprint: fp,
		SealKey:     seal,
	})
	if err != nil {
		t.Fatalf("RegisterOrReuse failed: %v", err)
	}
	st.Close()

	// 重新打开并验证数据完整恢复
	st2, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer st2.Close()

	recovered, err := st2.Get(node.ID)
	if err != nil {
		t.Fatalf("Get recovered failed: %v", err)
	}
	if recovered.Name != "persisted-node" || recovered.Address != "192.168.1.50" {
		t.Fatalf("Recovered node mismatch: %+v", recovered)
	}
}
