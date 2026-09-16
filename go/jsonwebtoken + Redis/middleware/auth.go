package middleware

import (
	"errors"
	"jwt-redis/config"
	"jwt-redis/services"
	"jwt-redis/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	CtxUserID  = "auth.userID"
	CtxTokenID = "auth.jti"
)

func Auth(cfg *config.Config, store *services.TokenStore) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		raw, ok := utils.BearerToken(ctx.GetHeader("Authorization"))
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing_token",
			})
			return
		}

		claims, err := utils.ParseAccessToken(cfg, raw)
		if err != nil {
			code := "invalid_token"
			if errors.Is(err, jwt.ErrTokenExpired) {
				code = "token_expired" // 让前端区分"该刷新了"和"token 是假的"
			}
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": code,
			})
			return
		}
		revoked, err := store.IsBlacklisted(ctx.Request.Context(), claims.ID)
		if err != nil {
			// fail-closed：Redis 挂了宁可拒绝，绝不放行可能已登出的 token
			ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "auth_backend_unavailable",
			})
		}
		if revoked {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token_revoked",
			})
			return
		}

		ctx.Set(CtxUserID, claims.UserID)
		ctx.Set(CtxTokenID, claims.ID)
		ctx.Next()
	}
}
