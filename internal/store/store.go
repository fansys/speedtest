// Package store 是节点数据的 SQLite 持久化仓储层。
//
// 基于标准库 database/sql 与官方标准 SQLite 驱动 (modernc.org/sqlite)，
// 具备 100% 完整的 SQLite 3 ACID 事务与持久化能力，且无需依赖外部 C 编译器或 libsqlite3 动态库，
// 支持在全平台与架构（Linux amd64/arm64、macOS、Windows、FreeBSD 等）上原生编译发布。
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound 表示指定 id 的节点不存在。
var ErrNotFound = errors.New("store: 节点不存在")

const timeLayout = time.RFC3339Nano

// Node 是一个测速节点。
type Node struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Address        string     `json:"address"`
	Port           int        `json:"port"`
	Protocol       string     `json:"protocol"`
	NodeKeyHash    string     `json:"node_key_hash"`
	KeyFingerprint string     `json:"key_fingerprint"`
	NodeKeySealed  *string    `json:"node_key_sealed,omitempty"`
	MetadataJSON   *string    `json:"metadata_json,omitempty"`
	Enabled        bool       `json:"enabled"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	LastStatus     string     `json:"last_status"`
	LastLatencyMs  *float64   `json:"last_latency_ms,omitempty"`
	LastError      *string    `json:"last_error,omitempty"`
}

// Store 是节点表的仓储；内部使用互斥锁对多步注册/更新业务决策进行串行化保护。
type Store struct {
	mu sync.Mutex
	db *sql.DB
}

// Open 打开（或创建）SQLite 数据库文件并执行数据迁移建表。
func Open(path string) (*Store, error) {
	var dsn string
	if path == ":memory:" {
		dsn = "file::memory:?cache=shared"
	} else {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("store: 无法创建数据目录 %s: %w", dir, err)
			}
		}
		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", path)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: 打开 SQLite 数据库失败: %w", err)
	}

	// 针对单个 SQLite 实例设置合理的连接池
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭底层 SQLite 数据库连接。
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
		);`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_key_hash ON nodes(node_key_hash);`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("store: 迁移失败: %w", err)
		}
	}
	return nil
}

const selectColumns = `id, name, address, port, protocol, node_key_hash, key_fingerprint,
	node_key_sealed, metadata_json, enabled, created_at, updated_at, last_seen_at,
	last_status, last_latency_ms, last_error`

func scanNode(scanner interface{ Scan(dest ...any) error }) (*Node, error) {
	n := &Node{}
	var enabledInt int
	var createdAtStr, updatedAtStr string
	var lastSeenAtStr sql.NullString
	var nodeKeySealed sql.NullString
	var metadataJSON sql.NullString
	var lastLatencyMs sql.NullFloat64
	var lastError sql.NullString

	err := scanner.Scan(
		&n.ID,
		&n.Name,
		&n.Address,
		&n.Port,
		&n.Protocol,
		&n.NodeKeyHash,
		&n.KeyFingerprint,
		&nodeKeySealed,
		&metadataJSON,
		&enabledInt,
		&createdAtStr,
		&updatedAtStr,
		&lastSeenAtStr,
		&n.LastStatus,
		&lastLatencyMs,
		&lastError,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	n.Enabled = enabledInt != 0
	if nodeKeySealed.Valid {
		v := nodeKeySealed.String
		n.NodeKeySealed = &v
	}
	if metadataJSON.Valid {
		v := metadataJSON.String
		n.MetadataJSON = &v
	}
	if lastLatencyMs.Valid {
		v := lastLatencyMs.Float64
		n.LastLatencyMs = &v
	}
	if lastError.Valid {
		v := lastError.String
		n.LastError = &v
	}

	if n.CreatedAt, err = time.Parse(timeLayout, createdAtStr); err != nil {
		return nil, fmt.Errorf("store: created_at 解析失败: %w", err)
	}
	if n.UpdatedAt, err = time.Parse(timeLayout, updatedAtStr); err != nil {
		return nil, fmt.Errorf("store: updated_at 解析失败: %w", err)
	}
	if lastSeenAtStr.Valid && lastSeenAtStr.String != "" {
		t, err := time.Parse(timeLayout, lastSeenAtStr.String)
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

	rows, err := s.db.Query(`SELECT ` + selectColumns + ` FROM nodes ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (s *Store) getLocked(id int64) (*Node, error) {
	row := s.db.QueryRow(`SELECT `+selectColumns+` FROM nodes WHERE id = ?`, id)
	return scanNode(row)
}

// Get 按 id 查找节点；不存在返回 ErrNotFound。
func (s *Store) Get(id int64) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(id)
}

func (s *Store) findByAddressPortLocked(address string, port int) (*Node, error) {
	row := s.db.QueryRow(`SELECT `+selectColumns+` FROM nodes WHERE address = ? AND port = ?`, address, port)
	n, err := scanNode(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return n, nil
}

func (s *Store) findByKeyHashLocked(hash string) (*Node, error) {
	row := s.db.QueryRow(`SELECT `+selectColumns+` FROM nodes WHERE node_key_hash = ?`, hash)
	n, err := scanNode(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return n, nil
}

// SetEnabled 启用/禁用节点。
func (s *Store) SetEnabled(id int64, enabled bool) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.getLocked(id); err != nil {
		return nil, err
	}

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	now := time.Now().UTC().Format(timeLayout)
	_, err := s.db.Exec(`UPDATE nodes SET enabled = ?, updated_at = ? WHERE id = ?`, enabledInt, now, id)
	if err != nil {
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
	_, err := s.db.Exec(`DELETE FROM nodes WHERE id = ?`, id)
	return err
}

// UpdateHealthResult 记录一次健康检查/测速之后的节点状态。
func (s *Store) UpdateHealthResult(id int64, status string, latencyMs *float64, errMsg *string, markSeen bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(timeLayout)
	var query string
	var args []any

	if markSeen {
		query = `UPDATE nodes SET last_status = ?, last_latency_ms = ?, last_error = ?, last_seen_at = ?, updated_at = ? WHERE id = ?`
		args = []any{status, latencyMs, errMsg, now, now, id}
	} else {
		query = `UPDATE nodes SET last_status = ?, last_latency_ms = ?, last_error = ?, updated_at = ? WHERE id = ?`
		args = []any{status, latencyMs, errMsg, now, id}
	}

	res, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RegisterParams 描述一次注册/自助注册所需的全部输入。
type RegisterParams struct {
	Name            string
	Address         string
	Port            int
	Protocol        string
	MetadataJSON    *string
	ExistingNodeKey *string

	GenerateKey func() string
	HashKey     func(string) string
	Fingerprint func(string) string
	SealKey     func(nodeKey string) (*string, error)
}

// RegisterOrReuse 是节点注册与 Key 复用的核心逻辑。
func (s *Store) RegisterOrReuse(p RegisterParams) (node *Node, nodeKey string, reused bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	target, err := s.findByAddressPortLocked(p.Address, p.Port)
	if err != nil {
		return nil, "", false, err
	}

	// 检查现有 Key 是否匹配同一节点
	if p.ExistingNodeKey != nil && *p.ExistingNodeKey != "" {
		hash := p.HashKey(*p.ExistingNodeKey)
		owner, err := s.findByKeyHashLocked(hash)
		if err != nil {
			return nil, "", false, err
		}

		if owner != nil && target != nil && owner.ID == target.ID {
			now := time.Now().UTC().Format(timeLayout)
			_, err := s.db.Exec(
				`UPDATE nodes SET name = ?, protocol = ?, metadata_json = ?, enabled = 1, updated_at = ? WHERE id = ?`,
				p.Name, p.Protocol, p.MetadataJSON, now, target.ID,
			)
			if err != nil {
				return nil, "", false, err
			}
			updated, err := s.getLocked(target.ID)
			if err != nil {
				return nil, "", false, err
			}
			return updated, *p.ExistingNodeKey, true, nil
		}
	}

	// 生成新 Key
	newKey := p.GenerateKey()
	hash := p.HashKey(newKey)
	fingerprint := p.Fingerprint(newKey)
	sealed, err := p.SealKey(newKey)
	if err != nil {
		return nil, "", false, err
	}

	now := time.Now().UTC().Format(timeLayout)

	if target == nil {
		res, err := s.db.Exec(
			`INSERT INTO nodes (name, address, port, protocol, node_key_hash, key_fingerprint, node_key_sealed, metadata_json, enabled, created_at, updated_at, last_status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, 'unknown')`,
			p.Name, p.Address, p.Port, p.Protocol, hash, fingerprint, sealed, p.MetadataJSON, now, now,
		)
		if err != nil {
			return nil, "", false, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, "", false, err
		}
		created, err := s.getLocked(id)
		if err != nil {
			return nil, "", false, err
		}
		return created, newKey, false, nil
	}

	// 更新已有节点并轮换 Key
	_, err = s.db.Exec(
		`UPDATE nodes SET name = ?, protocol = ?, metadata_json = ?, node_key_hash = ?, key_fingerprint = ?, node_key_sealed = ?, enabled = 1, updated_at = ? WHERE id = ?`,
		p.Name, p.Protocol, p.MetadataJSON, hash, fingerprint, sealed, now, target.ID,
	)
	if err != nil {
		return nil, "", false, err
	}
	updated, err := s.getLocked(target.ID)
	if err != nil {
		return nil, "", false, err
	}
	return updated, newKey, false, nil
}
