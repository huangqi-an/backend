package config

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env  string
	Port string

	RedisAddr     string
	RedisPassword string
	RedisDb       int
	KeyPrefix     string

	JWTSecret   []byte
	JWTIssuer   string
	JWTAudience string

	AccessTTL  time.Duration
	RefreshTTL time.Duration

	CookieSecure bool
	CookieDomain string
}

func (c *Config) IsProd() bool { return c.Env == "production" }

func Load() *Config {
	_ = godotenv.Load()

	c := &Config{
		Env:  env("APP_ENV", "development"),
		Port: env("PORT", "3001"),

		RedisAddr:     net.JoinHostPort(env("REDIS_HOST", "127.0.0.1"), env("REDIS_PORT", "6379")),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDb:       envInt("REDIS_DB", 0),
		KeyPrefix:     env("KEY_PREFIX", "go:auth:"),
		JWTSecret:     []byte(os.Getenv("JWT_SECRET")),
		JWTIssuer:     env("JWT_ISSUER", "backend-api"),
		JWTAudience:   env("JWT_AUDIENCE", "backend-web"),
		AccessTTL:     envDur("ACCESS_TTL", 15*time.Minute),
		RefreshTTL:    envDur("REFRESH_TTL", 30*24*time.Hour),
		CookieDomain:  os.Getenv("COOKIE_DOMAIN"),
	}

	c.CookieSecure = envBool("COOKIE_SECURE", c.IsProd())

	if len(c.JWTSecret) < 32 {
		log.Fatal("config: JWT_SECRET 至少 32 字节，用 openssl rand -base64 48 生成")
	}

	if c.RefreshTTL <= c.AccessTTL {
		log.Fatal("config: REFRESH_TTL 必须大于 ACCESS_TTL")
	}

	return c
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func envBool(k string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

func envDur(k string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil {
		return v
	}
	return def
}

// loadEnvFiles 按优先级从低到高加载多个 .env 文件，并向上搜索父目录。
// 优先级（高 → 低）：.env.local > .env.<APP_ENV> > .env；越靠近项目的目录越优先。
func loadEnvFiles() {
	names := []string{".env", ".env." + os.Getenv("APP_ENV"), ".env.local"}

	for _, name := range names {
		if strings.HasSuffix(name, ".") { // APP_ENV 为空时跳过 ".env."
			continue
		}
		paths := findUpAll(name) // 从根到当前目录，由远及近
		for _, p := range paths {
			if err := godotenv.Overload(p); err != nil && !os.IsNotExist(err) {
				log.Printf("config: 加载 %s 失败: %v", p, err)
			}
		}
	}
}

// findUpAll 从当前工作目录向上收集所有名为 name 的文件，返回从最外层到最内层的路径。
func findUpAll(name string) []string {
	dir, err := os.Getwd()
	if err != nil {
		return nil
	}
	var paths []string
	for {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			paths = append(paths, p)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// 反转：从最外层（根）到当前目录
	for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
		paths[i], paths[j] = paths[j], paths[i]
	}
	return paths
}
