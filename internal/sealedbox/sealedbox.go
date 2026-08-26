// Package sealedbox 用主密钥（SECRET_KEY）封存 node_key。
//
// 背景：需求要求 node_key「只存哈希」，但同时要求中心服务能主动用节点 key 去访问
// 节点（健康检查 / 发起测速）。哈希不可逆，因此这两点无法只靠哈希同时满足。
//
// 本模块提供第二条轨道：校验永远只依赖 sha256 哈希（见 internal/security）；若启用
// 了 node_key 密封存储，额外保存一份用 SECRET_KEY 加密封存的密文，使中心可以自动
// 巡检。密文离开 SECRET_KEY 不可解，数据库泄露本身不会泄露 node_key。
//
// 算法：AES-256-GCM，密钥取 SHA-256(secretKey)。SECRET_KEY 本身要求足够长（>=16
// 字符，实践中应远高于此），用它的哈希做 AES 密钥是标准做法，无需额外引入 KDF 依赖。
package sealedbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// ErrSealed 在密文损坏、被篡改，或 SECRET_KEY 与封存时不一致时返回。
var ErrSealed = errors.New("sealedbox: 封存数据校验失败")

func deriveKey(secretKey string) []byte {
	sum := sha256.Sum256([]byte(secretKey))
	return sum[:]
}

// Seal 加密封存，返回可直接入库的 base64 字符串。
func Seal(secretKey, plaintext string) (string, error) {
	if secretKey == "" {
		return "", fmt.Errorf("sealedbox: SECRET_KEY 为空，无法封存 node_key")
	}
	block, err := aes.NewCipher(deriveKey(secretKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Unseal 解封；密文被篡改或密钥不对时返回 ErrSealed。
func Unseal(secretKey, sealed string) (string, error) {
	if secretKey == "" {
		return "", fmt.Errorf("sealedbox: SECRET_KEY 为空，无法解封 node_key")
	}
	blob, err := base64.URLEncoding.DecodeString(sealed)
	if err != nil {
		return "", fmt.Errorf("%w: 不是合法的 base64", ErrSealed)
	}
	block, err := aes.NewCipher(deriveKey(secretKey))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return "", fmt.Errorf("%w: 数据长度不足", ErrSealed)
	}
	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w (SECRET_KEY 不匹配或数据被篡改)", ErrSealed)
	}
	return string(plaintext), nil
}
