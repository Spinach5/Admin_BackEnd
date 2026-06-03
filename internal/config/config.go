package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	GinMode        string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	JWTSecret      string
	JWTExpireHours string
	FrontendURLs   []string // 改为切片，支持多个前端源
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("警告：加载 .env 文件失败:", err)
	}
	return &Config{
		Port:           getEnv("PORT", "3001"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "name"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "dbname"),
		JWTSecret:      getEnv("JWT_SECRET", "default-secret"),
		JWTExpireHours: getEnv("JWT_EXPIRE_HOURS", "24"),
		FrontendURLs:   getEnvAsSlice("FRONTEND_URLS", []string{}), // 调用切片解析函数
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvAsSlice 读取环境变量中的逗号分隔值，返回去除空格的字符串切片
func getEnvAsSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts
	}
	return fallback
}
