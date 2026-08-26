// Package security 提供令牌生成、哈希与恒定时间校验。
//
// 约定：
//   - 所有令牌只通过请求头传递（X-Admin-Token / X-Registration-Token / X-Node-Key
//     或 Authorization: Bearer），绝不接受 query string，避免进入访问日志。
//   - node_key 只存 sha256 哈希；比较一律使用恒定时间算法。
package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// NodeKeyBytes 是 node_key 的随机字节数（256 bit 熵）。
const NodeKeyBytes = 32

// MinNodeKeyLength 是接受外部提交的 node_key 时要求的最小长度。
const MinNodeKeyLength = 24

// GenerateNodeKey 生成高熵 node_key（等价于 Python secrets.token_urlsafe(32)）。
func GenerateNodeKey() string {
	buf := make([]byte, NodeKeyBytes)
	if _, err := rand.Read(buf); err != nil {
		panic("security: 无法读取随机数: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// HashNodeKey 返回 node_key 的 sha256 十六进制摘要。
//
// node_key 本身是 256bit 均匀随机值，不存在字典/暴力风险，因此使用快速哈希而非 KDF。
func HashNodeKey(nodeKey string) string {
	sum := sha256.Sum256([]byte(nodeKey))
	return hex.EncodeToString(sum[:])
}

// KeyFingerprint 返回可安全展示的短指纹，用于在 UI 上区分 key，不可反推原文。
func KeyFingerprint(nodeKey string) string {
	h := HashNodeKey(nodeKey)
	if len(h) < 12 {
		return h
	}
	return h[:12]
}

// ConstantTimeEquals 用恒定时间算法比较两个字符串。
func ConstantTimeEquals(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

// ExtractBearer 从 Authorization 头里取出 Bearer 令牌；格式不对返回空字符串。
func ExtractBearer(authorization string) string {
	if authorization == "" {
		return ""
	}
	scheme, value, found := strings.Cut(authorization, " ")
	if !found || !strings.EqualFold(scheme, "bearer") {
		return ""
	}
	value = strings.TrimSpace(value)
	return value
}

// PickToken 优先使用专用请求头的值，否则回退到 Authorization: Bearer。
func PickToken(headerValue, authorization string) string {
	if v := strings.TrimSpace(headerValue); v != "" {
		return v
	}
	return ExtractBearer(authorization)
}
