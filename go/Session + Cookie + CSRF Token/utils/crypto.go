package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
)

// GenerateCsrfToken 函数生成一个32字节的随机令牌，并对其进行base64url编码。
// 等同于 Node 的 crypto.randomBytes(32).toString("base64url")。
func GenerateCsrfToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SafeEqual在常数时间内比较两个字符串，以防止时序攻击。
// 等同于 Node 的 crypto.timingSafeEqual。
func SafeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
