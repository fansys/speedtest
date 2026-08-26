// Package store 是节点数据的 SQLite 持久化层。
//
// 安全约定：
//   - node_key_hash 是 node_key 的 sha256，用于校验，不可反推。
//   - node_key_sealed 是可选的加密封存副本（依赖 SECRET_KEY），仅供中心服务主动
//     访问节点时解封使用，任何 API 都不会返回它。
//   - key_fingerprint 是哈希前 12 位，可安全展示，用于人工区分 key。
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"librespeed-service/internal/sqlite3"
)

// ErrNotFound 表示指定 id 的节点不存在。
var ErrNotFound = errors.New("store: 节点不存在")

const timeLayout = time.RFC3339Nano

// Node 是一个测速节点。
type Node struct {
	ID             int64
	Name           string
	Address        string
	Port           int
	Protocol       string
	NodeKeyHash    string
	KeyFingerprint string
	NodeKeySealed  *string
	MetadataJSON   *string
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastSeenAt     *time.Time
	LastStatus     string
	LastLatencyMs  *float64
	LastError      *string
}

// Store 是节点表的仓储；内部用一把互斥锁把「读取当前状态 + 决策 + 写回」的多步业务
// 操作串行化，避免并发注册/删除等操作交叉产生不一致（单条 SQL 层面的一致性由
// SQLite 自身保证，但跨多条语句的业务决策需要在 Go 侧再加一层锁）。
type Store struct {
	mu sync.Mutex
	db *sqlite3.DB
}

// Open 打开（或创建）SQLite 数据库文件并完成建表迁移。
func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("store: 无法创建数据目录 %s: %w", dir, err)
			}
		}
	}
	db, err := sqlite3.Open(path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭底层数据库连接。
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS nodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			address TEXT NOT NULL,
			port INTEGER NOT NULL,
			protocol TEXT NOT NULL DEFAULT 'http',
			node_key_hash TEXT NOT NULL,
			key_fingerprint TEXT NOT NULL,
			node_key_sealed TEXT,
			metadata_json TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_seen_at TEXT,
			last_status TEXT NOT NULL DEFAULT 'unknown',
			last_latency_ms REAL,
			last_error TEXT,
			UNIQUE(address, port)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_key_hash ON nodes(node_key_hash)`,
	}
	for _, stmt := range stmts {
		if err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("store: 迁移失败: %w", err)
		}
	}
	return nil
}

const selectColumns = `id, name, address, port, protocol, node_key_hash, key_fingerprint,
	node_key_sealed, metadata_json, enabled, created_at, updated_at, last_seen_at,
	last_status, last_latency_ms, last_error`

func scanNode(stmt *sqlite3.Stmt) (*Node, error) {
	n := &Node{
		ID:             stmt.ColumnInt64(0),
		Name:           stmt.ColumnText(1),
		Address:        stmt.ColumnText(2),
		Port:           int(stmt.ColumnInt64(3)),
		Protocol:       stmt.ColumnText(4),
		NodeKeyHash:    stmt.ColumnText(5),
		KeyFingerprint: stmt.ColumnText(6),
		NodeKeySealed:  stmt.ColumnNullableText(7),
		MetadataJSON:   stmt.ColumnNullableText(8),
		Enabled:        stmt.ColumnInt64(9) != 0,
		LastStatus:     stmt.ColumnText(13),
		LastLatencyMs:  stmt.ColumnNullableDouble(14),
		LastError:      stmt.ColumnNullableText(15),
	}
	var err error
	if n.CreatedAt, err = time.Parse(timeLayout, stmt.ColumnText(10)); err != nil {
		return nil, fmt.Errorf("store: created_at 解析失败: %w", err)
	}
	if n.UpdatedAt, err = time.Parse(timeLayout, stmt.ColumnText(11)); err != nil {
		return nil, fmt.Errorf("store: updated_at 解析失败: %w", err)
	}
	if raw := stmt.ColumnNullableText(12); raw != nil {
		t, err := time.Parse(timeLayout, *raw)
		if err != nil {
			return nil, fmt.Errorf("store: last_seen_at 解析失败: %w", err)
		}
		n.LastSeenAt = &t
	}
	return n, nil
}

// List 返回全部节点，按 id 升序。
func (s *Store) List() ([]*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stmt, err := s.db.Prepare(`SELECT ` + selectColumns + ` FROM nodes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer stmt.Finalize()

	var nodes []*Node
	for {
		hasRow, err := stmt.Step()
		if err != nil {
			return nil, err
		}
		if !hasRow {
			break
		}
		n, err := scanNode(stmt)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (s *Store) getLocked(id int64) (*Node, error) {
	stmt, err := s.db.Prepare(`SELECT ` + selectColumns + ` FROM nodes WHERE id = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Finalize()
	if err := stmt.BindInt64(1, id); err != nil {
		return nil, err
	}
	hasRow, err := stmt.Step()
	if err != nil {
		return nil, err
	}
	if !hasRow {
		return nil, ErrNotFound
	}
	return scanNode(stmt)
}

// Get 按 id 查找节点；不存在返回 ErrNotFound。
func (s *Store) Get(id int64) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(id)
}

func (s *Store) findByAddressPortLocked(address string, port int) (*Node, error) {
	stmt, err := s.db.Prepare(`SELECT ` + selectColumns + ` FROM nodes WHERE address = ? AND port = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Finalize()
	if err := stmt.BindText(1, address); err != nil {
		return nil, err
	}
	if err := stmt.BindInt64(2, int64(port)); err != nil {
		return nil, err
	}
	hasRow, err := stmt.Step()
	if err != nil {
		return nil, err
	}
	if !hasRow {
		return nil, nil
	}
	return scanNode(stmt)
}

func (s *Store) findByKeyHashLocked(hash string) (*Node, error) {
	stmt, err := s.db.Prepare(`SELECT ` + selectColumns + ` FROM nodes WHERE node_key_hash = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Finalize()
	if err := stmt.BindText(1, hash); err != nil {
		return nil, err
	}
	hasRow, err := stmt.Step()
	if err != nil {
		return nil, err
	}
	if !hasRow {
		return nil, nil
	}
	return scanNode(stmt)
}

// SetEnabled 启用/禁用节点。
func (s *Store) SetEnabled(id int64, enabled bool) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.getLocked(id); err != nil {
		return nil, err
	}
	stmt, err := s.db.Prepare(`UPDATE nodes SET enabled = ?, updated_at = ? WHERE id = ?`)
	if err != nil {
		return nil, err
	}
	defer stmt.Finalize()
	stmt.BindInt64(1, boolToInt(enabled))
	stmt.BindText(2, nowString())
	stmt.BindInt64(3, id)
	if _, err := stmt.Step(); err != nil {
		return nil, err
	}
	return s.getLocked(id)
}

// Delete 删除节点。
func (s *Store) Delete(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.getLocked(id); err != nil {
		return err
	}
	stmt, err := s.db.Prepare(`DELETE FROM nodes WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Finalize()
	stmt.BindInt64(1, id)
	_, err = stmt.Step()
	return err
}

// UpdateHealthResult 记录一次健康检查/测速之后的节点状态。
func (s *Store) UpdateHealthResult(id int64, status string, latencyMs *float64, errMsg *string, markSeen bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stmt, err := s.db.Prepare(`UPDATE nodes SET last_status = ?, last_latency_ms = ?, last_error = ?,
		last_seen_at = CASE WHEN ? THEN ? ELSE last_seen_at END, updated_at = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Finalize()
	stmt.BindText(1, status)
	stmt.BindNullableDouble(2, latencyMs)
	stmt.BindNullableText(3, errMsg)
	stmt.BindInt64(4, boolToInt(markSeen))
	stmt.BindText(5, nowString())
	stmt.BindText(6, nowString())
	stmt.BindInt64(7, id)
	_, err = stmt.Step()
	return err
}

// RegisterParams 描述一次注册/自助注册所需的全部输入。密钥生成/哈希/封存都通过
// 回调传入，store 包本身不依赖 internal/security、internal/sealedbox，避免不必要
// 的耦合，也方便单测里注入固定的 key 生成函数。
type RegisterParams struct {
	Name            string
	Address         string // 已经过 netguard 校验/归一化的 host
	Port            int
	Protocol        string
	MetadataJSON    *string
	ExistingNodeKey *string // 为空指针或空字符串表示没有携带旧 key

	GenerateKey func() string
	HashKey     func(string) string
	Fingerprint func(string) string
	SealKey     func(nodeKey string) (*string, error) // 返回 nil 表示不封存
}

// RegisterOrReuse 是节点注册的核心逻辑，供手动 /api/register 与自助
// /api/register/auto 复用。
//
// 以 address+port 作为节点身份。ExistingNodeKey 只有在它确实属于「当前这个
// address+port 对应的节点」时才会被复用（不轮换）；否则一律生成新 key 并（视情况）
// 创建或轮换该 address+port 上的节点——绝不会去改动 key 实际归属的另一个节点，
// 防止旧 key 被用来冒领/覆盖别的地址端口。
func (s *Store) RegisterOrReuse(p RegisterParams) (node *Node, nodeKey string, reused bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	target, err := s.findByAddressPortLocked(p.Address, p.Port)
	if err != nil {
		return nil, "", false, err
	}

	if p.ExistingNodeKey != nil && *p.ExistingNodeKey != "" {
		hash := p.HashKey(*p.ExistingNodeKey)
		owner, err := s.findByKeyHashLocked(hash)
		if err != nil {
			return nil, "", false, err
		}
		if owner != nil && target != nil && owner.ID == target.ID {
			if err := s.updateNodeLocked(target.ID, p.Name, p.Protocol, p.MetadataJSON, nil, nil, nil, true); err != nil {
				return nil, "", false, err
			}
			updated, err := s.getLocked(target.ID)
			if err != nil {
				return nil, "", false, err
			}
			return updated, *p.ExistingNodeKey, true, nil
		}
	}

	newKey := p.GenerateKey()
	hash := p.HashKey(newKey)
	fingerprint := p.Fingerprint(newKey)
	sealed, err := p.SealKey(newKey)
	if err != nil {
		return nil, "", false, err
	}

	if target == nil {
		id, err := s.insertNodeLocked(p.Name, p.Address, p.Port, p.Protocol, hash, fingerprint, sealed, p.MetadataJSON)
		if err != nil {
			return nil, "", false, err
		}
		created, err := s.getLocked(id)
		if err != nil {
			return nil, "", false, err
		}
		return created, newKey, false, nil
	}

	if err := s.updateNodeLocked(target.ID, p.Name, p.Protocol, p.MetadataJSON, &hash, &fingerprint, sealed, true); err != nil {
		return nil, "", false, err
	}
	updated, err := s.getLocked(target.ID)
	if err != nil {
		return nil, "", false, err
	}
	return updated, newKey, false, nil
}

func (s *Store) insertNodeLocked(name, address string, port int, protocol, hash, fingerprint string, sealed, metadataJSON *string) (int64, error) {
	stmt, err := s.db.Prepare(`INSERT INTO nodes
		(name, address, port, protocol, node_key_hash, key_fingerprint, node_key_sealed, metadata_json,
		 enabled, created_at, updated_at, last_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, 'unknown')`)
	if err != nil {
		return 0, err
	}
	defer stmt.Finalize()

	now := nowString()
	stmt.BindText(1, name)
	stmt.BindText(2, address)
	stmt.BindInt64(3, int64(port))
	stmt.BindText(4, protocol)
	stmt.BindText(5, hash)
	stmt.BindText(6, fingerprint)
	stmt.BindNullableText(7, sealed)
	stmt.BindNullableText(8, metadataJSON)
	stmt.BindText(9, now)
	stmt.BindText(10, now)

	if _, err := stmt.Step(); err != nil {
		if errors.Is(err, sqlite3.ErrConstraint) {
			return 0, fmt.Errorf("store: address+port 已存在: %w", err)
		}
		return 0, err
	}
	return s.db.LastInsertRowID(), nil
}

// updateNodeLocked 更新一个已存在节点。hash/fingerprint/sealed 为 nil 表示保持不变
// （复用旧 key 场景）；不为 nil 则连同 key 一起轮换。
func (s *Store) updateNodeLocked(id int64, name, protocol string, metadataJSON, hash, fingerprint, sealed *string, enabled bool) error {
	if hash != nil {
		stmt, err := s.db.Prepare(`UPDATE nodes SET name = ?, protocol = ?, metadata_json = ?,
			node_key_hash = ?, key_fingerprint = ?, node_key_sealed = ?, enabled = ?, updated_at = ?
			WHERE id = ?`)
		if err != nil {
			return err
		}
		defer stmt.Finalize()
		stmt.BindText(1, name)
		stmt.BindText(2, protocol)
		stmt.BindNullableText(3, metadataJSON)
		stmt.BindText(4, *hash)
		stmt.BindText(5, *fingerprint)
		stmt.BindNullableText(6, sealed)
		stmt.BindInt64(7, boolToInt(enabled))
		stmt.BindText(8, nowString())
		stmt.BindInt64(9, id)
		_, err = stmt.Step()
		return err
	}

	stmt, err := s.db.Prepare(`UPDATE nodes SET name = ?, protocol = ?, metadata_json = ?, enabled = ?, updated_at = ?
		WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Finalize()
	stmt.BindText(1, name)
	stmt.BindText(2, protocol)
	stmt.BindNullableText(3, metadataJSON)
	stmt.BindInt64(4, boolToInt(enabled))
	stmt.BindText(5, nowString())
	stmt.BindInt64(6, id)
	_, err = stmt.Step()
	return err
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func nowString() string {
	return time.Now().UTC().Format(timeLayout)
}
