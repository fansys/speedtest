package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NodeState 是节点自助注册后持久化到 node.ini [node] 段里的键值对。
type NodeState map[string]string

// LoadNodeState 读取 node.ini 的 [node] 段；文件不存在或没有该段时返回 nil。
func LoadNodeState(path string) (NodeState, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("config: 无法打开 %s: %w", path, err)
	}
	defer f.Close()

	state := NodeState{}
	inSection := false
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			inSection = line == "[node]"
			if inSection {
				found = true
			}
			continue
		}
		if !inSection {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		state[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("config: 读取 %s 失败: %w", path, err)
	}
	if !found {
		return nil, nil
	}
	return state, nil
}

// SaveNodeState 把 fields 原子写入 node.ini 的 [node] 段（临时文件 + rename），
// 权限收紧到 0600；空字符串的字段会被跳过（与 Python 版 node_state.py 行为一致）。
func SaveNodeState(path string, fields map[string]string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("config: 无法创建目录 %s: %w", dir, err)
		}
	}

	keys := make([]string, 0, len(fields))
	for k, v := range fields {
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("[node]\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%s = %s\n", k, fields[k])
	}

	tmp, err := os.CreateTemp(dir, ".node-ini-*.tmp")
	if err != nil {
		return fmt.Errorf("config: 无法创建临时文件: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(b.String()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("config: 写入临时文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: 关闭临时文件失败: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: 设置临时文件权限失败: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config: 原子替换 %s 失败: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("config: 设置 %s 权限失败: %w", path, err)
	}
	return nil
}
