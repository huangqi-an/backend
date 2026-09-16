package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"jwt-redis/config"
	"strings"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// 48 字节随机 → 384bit 熵
func NewRefreshToken() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "v1." + base64.RawURLEncoding.EncodeToString(b), nil
}

// 直接用 SHA-256 做摘要即可：原文是高熵随机串，不存在字典攻击。
// 不要用 bcrypt/argon2，会白白给每次刷新加几十毫秒。
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// 签发 Access Token：jti 用于黑名单，sub/uid 是业务身份
func NewAccessToken(cfg *config.Config, userId string) (raw, jti string, expiresAt time.Time, err error) {
	now := time.Now()
	jti = uuid.New().String()
	expiresAt = now.Add(cfg.AccessTTL)

	claims := AccessClaims{
		UserID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userId,
			Issuer:    cfg.JWTIssuer,
			Audience:  jwt.ClaimStrings{cfg.JWTAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	raw, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(cfg.JWTSecret)
	return raw, jti, expiresAt, err
}

// 校验端必须锁定算法白名单，否则存在 alg 混淆 / alg=none 绕过
func ParseAccessToken(cfg *config.Config, raw string) (*AccessClaims, error) {
	claims := new(AccessClaims)
	_, err := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(cfg.JWTIssuer),
		jwt.WithAudience(cfg.JWTAudience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	).ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return cfg.JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}
	if claims.UserID == "" {
		return nil, errors.New("token missing uid")
	}
	return claims, nil
}

// 登出时用：签名照样验，但不验过期，这样才能拿到已过期 access 的 jti 去拉黑
func ParseAccessTokenIgnoringExpiry(cfg *config.Config, raw string) (*AccessClaims, error) {
	claims := new(AccessClaims)
	_, err := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithoutClaimsValidation(),
	).ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		return cfg.JWTSecret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func BearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}
