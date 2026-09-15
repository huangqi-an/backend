package routes

import (
	"net/http"
	"session-csrf/config"
	"session-csrf/middleware"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes 挂载了 POST /api/login 和 POST /api/logout 路由。
// 这些是在CsrfGuard之后注册的，因此它们受到CSRF保护。
//
// 相当于 Express 的 authRoutes：
// authRoutes.post("/api/login", ...)
// authRoutes.post("/api/logout", ...)
func RegisterAuthRoutes(r *gin.Engine, cfg *config.Config) {
	r.POST("/api/login", loginHandler(cfg))
	r.POST("/api/logout", logoutHandler(cfg))
}

func loginHandler(cfg *config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)

		// 演示：硬编码用户信息；真实应用会在此处验证凭证
		session.Set("userId", "u1")
		// 登录后重新生成CSRF令牌以防止会话固定攻击
		newToken, err := middleware.RegenerateCsrfToken(ctx)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "token regeneration failed"})
			return
		}
		// 在RegenerateCsrfToken内部已经调用了session.Save()，
		// 但userId是在之后设置的——需要再次保存
		if err := session.Save(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "session save failed"})
		}

		ctx.JSON(http.StatusOK, gin.H{"ok": true, "csrfToken": newToken})
	}
}

func logoutHandler(cfg *config.Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)
		// 清除所有会话数据（等同于req.session.destroy()）
		session.Clear()
		session.Options(sessions.Options{
			Path:     "/",
			MaxAge:   -1, // expires immediately
			HttpOnly: true,
		})
		if err := session.Save(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
			return
		}
		// 清除浏览器Cookie — 等同于 res.clearCookie(name, options)
		ctx.SetCookie(cfg.SessionName, "", -1, "/", "", cfg.IsProd, true)
		ctx.JSON(http.StatusOK, gin.H{
			"ok":      true,
			"message": "logged out",
		})
	}
}
