package routes

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"jwt-redis/config"
	"jwt-redis/middleware"
	"jwt-redis/services"
	"jwt-redis/utils"
)

const refreshCookieName = "rt"

type AuthHandler struct {
	cfg   *config.Config
	store *services.TokenStore
}

func NewAuthHandler(cfg *config.Config, store *services.TokenStore) *AuthHandler {
	return &AuthHandler{cfg: cfg, store: store}
}

func (h *AuthHandler) Register(r gin.IRouter, guard gin.HandlerFunc) {
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)

	api := r.Group("/api", guard)
	api.GET("/me", h.Me)
}

func (h *AuthHandler) setRefreshCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/auth", // 只在 /auth/* 发送，业务接口拿不到这个 cookie
		Domain:   h.cfg.CookieDomain,
		MaxAge:   int(h.cfg.RefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// 清理 cookie 时属性必须和写入时一致，否则浏览器不会覆盖掉
func (h *AuthHandler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}

	// TODO 换成真实用户表：argon2id/bcrypt 校验 + 登录失败限流（按 IP 和账号双维度）
	userID := ""
	if req.Username == "admin" && req.Password == "123456" {
		userID = "u_1"
	}
	if userID == "" {
		// 统一错误码，别区分"用户不存在/密码错误"，避免账号枚举
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	ctx := c.Request.Context()
	refresh, err := utils.NewRefreshToken()
	if err != nil {
		slog.Error("gen refresh token", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	familyID := uuid.NewString() // 每次登录开一条新链
	if err := h.store.SaveRefresh(ctx, utils.HashRefreshToken(refresh), userID, familyID); err != nil {
		slog.Error("save refresh", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	access, _, _, err := utils.NewAccessToken(h.cfg, userID)
	if err != nil {
		slog.Error("sign access", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	h.setRefreshCookie(c, refresh)
	c.JSON(http.StatusOK, gin.H{
		"accessToken": access,
		"tokenType":   "Bearer",
		"expiresIn":   int(h.cfg.AccessTTL.Seconds()),
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(refreshCookieName)
	if err != nil || raw == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no_refresh_token"})
		return
	}

	ctx := c.Request.Context()
	oldHash := utils.HashRefreshToken(raw)

	rec, err := h.store.ActiveRecord(ctx, oldHash)
	if err != nil {
		if !errors.Is(err, services.ErrNotFound) {
			slog.Error("read refresh record", "err", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth_backend_unavailable"})
			return
		}
		// 活跃记录不在：查是不是"已被轮换出去的旧 token"
		if fid, _ := h.store.UsedFamily(ctx, oldHash); fid != "" {
			if _, err := h.store.RevokeFamily(ctx, fid); err != nil {
				slog.Error("revoke family", "err", err)
			}
			slog.Warn("refresh token reuse detected", "family", fid)
			h.clearRefreshCookie(c)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh_reuse_detected"})
			return
		}
		h.clearRefreshCookie(c)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_refresh_token"})
		return
	}

	// 先签发 access 再轮换：签名失败时不会白白消耗掉客户端唯一的 refresh token
	access, _, _, err := utils.NewAccessToken(h.cfg, rec.UserID)
	if err != nil {
		slog.Error("sign access", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	newRaw, err := utils.NewRefreshToken()
	if err != nil {
		slog.Error("gen refresh token", "err", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	res, err := h.store.Rotate(ctx, oldHash, utils.HashRefreshToken(newRaw), rec.UserID, rec.FamilyID)
	if err != nil {
		slog.Error("rotate refresh", "err", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "auth_backend_unavailable"})
		return
	}

	switch res {
	case services.RotateReused:
		if _, err := h.store.RevokeFamily(ctx, rec.FamilyID); err != nil {
			slog.Error("revoke family", "err", err)
		}
		slog.Warn("refresh token reuse detected", "family", rec.FamilyID, "user", rec.UserID)
		h.clearRefreshCookie(c)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "refresh_reuse_detected"})
		return
	case services.RotateInvalid:
		h.clearRefreshCookie(c)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_refresh_token"})
		return
	}

	h.setRefreshCookie(c, newRaw)
	c.JSON(http.StatusOK, gin.H{
		"accessToken": access,
		"tokenType":   "Bearer",
		"expiresIn":   int(h.cfg.AccessTTL.Seconds()),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	ctx := c.Request.Context()

	if raw, err := c.Cookie(refreshCookieName); err == nil && raw != "" {
		hash := utils.HashRefreshToken(raw)
		if rec, err := h.store.ActiveRecord(ctx, hash); err == nil {
			if err := h.store.RevokeOne(ctx, hash, rec.FamilyID); err != nil {
				slog.Error("revoke refresh", "err", err)
			}
		}
	}

	// 让当前 access 也立刻失效；如果接受"access 最长 15 分钟后自然过期"，这段可以删掉
	if raw, ok := utils.BearerToken(c.GetHeader("Authorization")); ok {
		if claims, err := utils.ParseAccessTokenIgnoringExpiry(h.cfg, raw); err == nil && claims.ExpiresAt != nil {
			if ttl := time.Until(claims.ExpiresAt.Time); ttl > 0 {
				if err := h.store.BlacklistAccess(ctx, claims.ID, ttl); err != nil {
					slog.Error("blacklist access", "err", err)
				}
			}
		}
	}

	h.clearRefreshCookie(c)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get(middleware.CtxUserID)
	c.JSON(http.StatusOK, gin.H{"userId": userID})
}
