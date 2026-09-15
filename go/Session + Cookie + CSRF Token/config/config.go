package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	NodeEnv       string
	Port          string
	SessionSecret string
	SessionName   string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	SessionTTL    int
	IsProd        bool
}

// 加载并读取 .env 文件（如果存在）和环境变量，验证其是否符合要求
// 解析字段，并返回一个强类型的配置对象。如果缺少必需变量，则 panic。
func Load() *Config {
	_ = godotenv.Load()
	c := &Config{
		NodeEnv:       getEnv("NODE_ENV", "development"),
		Port:          getEnv("PORT", "3000"),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		SessionName:   getEnv("SESSION_NAME", "sid"),
		RedisHost:     getEnv("REDIS_HOST", "127.0.0.1"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		SessionTTL:    getEnvInt("SESSION_TTL", 86400),
	}
	c.IsProd = c.NodeEnv == "production"
	if c.SessionSecret == "" {
		panic("SESSION_SECRET is required")
	}
	return c
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			panic(fmt.Sprintf("invalid int for %s: %s", key, v))
		}
		return n
	}
	return fallback
}
