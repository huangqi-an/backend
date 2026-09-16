package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"jwt-redis/config"
	"jwt-redis/middleware"
	redisclient "jwt-redis/redis"
	"jwt-redis/routes"
	"jwt-redis/services"
)

func main() {
	cfg := config.Load()
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	rdb, err := redisclient.New(cfg)
	if err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	store := services.NewTokenStore(rdb, cfg.KeyPrefix, cfg.RefreshTTL)
	auth := routes.NewAuthHandler(cfg, store)

	r := gin.New()
	r.Use(gin.Recovery())
	// 没有反向代理时必须关掉，否则 X-Forwarded-For 可伪造，限流会被绕过
	_ = r.SetTrustedProxies(nil)

	r.GET("/healthz", func(c *gin.Context) {
		if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "redis": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	auth.Register(r, middleware.Auth(cfg, store))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
}
