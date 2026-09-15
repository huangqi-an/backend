package middleware

import (
	"log"
	"net"
	"net/http"
	"session-csrf/config"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

var (
	sessionStore sessions.Store
	storeOnce    sync.Once
	storeErr     error
)

func getSessionStore(cfg *config.Config) (sessions.Store, error) {
	storeOnce.Do(
		func() {
			sessionStore, storeErr = redis.NewStore(
				10,
				"tcp",
				// cfg.RedisHost+":"+cfg.RedisPort,
				net.JoinHostPort(cfg.RedisHost, cfg.RedisPort),
				"",
				cfg.RedisPassword,
				[]byte(cfg.SessionSecret),
			)
			if storeErr != nil {
				return
			}
			sessionStore.Options(sessions.Options{
				Path:     "/",
				MaxAge:   cfg.SessionTTL,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   cfg.IsProd,
			})
		},
	)
	return sessionStore, storeErr
}

// SessionMiddleware 创建一个基于 Redis 的会话存储，并返回一个 Gin
// 中间件。商店使用SESSION_SECRET来对会话cookie进行签名。
//
// 等同于 Express 的 buildSessionMiddleware()：
// - connect-redis -> gin-contrib/sessions/redis
// - 会话密钥 -> keyPairs[0]
// - cookie 选项 -> store.Options()
/**
func SessionMiddleware(cfg *config.Config) gin.HandlerFunc {
	store, err := redis.NewStore(
		10,
		"tcp",
		cfg.RedisHost+":"+cfg.RedisPort,
		cfg.RedisPassword,
		cfg.SessionSecret,
	)
	if err != nil {
		log.Fatal("failed to create redis session store: ", err)
	}
	// Cookie属性 — 镜像express-session cookie配置
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   cfg.SessionTTL,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.IsProd,
	})
	// cfg.SessionName 是 cookie 的名称（等同于 express-session 的 name 选项）
	return sessions.Sessions(cfg.SessionName, store)
}
*/
func SessionMiddleware(cfg *config.Config) gin.HandlerFunc {
	store, err := getSessionStore(cfg)
	if err != nil {
		log.Fatal("failed to create redis session store: ", err)
	}
	return sessions.Sessions(cfg.SessionName, store)
}
