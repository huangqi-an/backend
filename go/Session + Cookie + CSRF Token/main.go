package main

import (
	"log"
	"net/http"
	"session-csrf/config"
	"session-csrf/middleware"
	"session-csrf/routes"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// 在生产环境中，切换到 gin.Release 模式以提高性能
	if cfg.IsProd {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	//1. 会话中间件——必须优先，因为跨站点请求伪造（CSRF）依赖它
	r.Use(middleware.SessionMiddleware(cfg))

	//2. 令牌端点 — 必须位于 CsrfGuard 之前
	routes.RegisterCsrfRoutes(r)

	//3. CSRF防护 — 阻止没有有效令牌的不安全方法
	r.Use(middleware.CsrfGuard())

	//4. 健康检查（GET，不受CSRF保护影响）
	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	//5. 业务路由（POST — 受CSRF保护）
	routes.RegisterAuthRoutes(r, cfg)

	// 演示路由：需要有效的会话 + CSRF 令牌
	r.POST("/api/danger", func(c *gin.Context) {
		session := sessions.Default(c)
		userId, _ := session.Get("userId").(string)
		c.JSON(http.StatusOK, gin.H{"ok": true, "userId": userId})
	})
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("failed to start server: ", err)
	}
}
