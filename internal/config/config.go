package config

import (
	"os"

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
	FrontendURL    string
}

func Load() *Config {
	godotenv.Load()

	return &Config{
		Port:           getEnv("PORT", "3001"),
		GinMode:        getEnv("GIN_MODE", "debug"),
		DBHost:         getEnv("DB_HOST", "defalutHost"),
		DBPort:         getEnv("DB_PORT", "3306"),
		DBUser:         getEnv("DB_USER", "name"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "dbname"),
		JWTSecret:      getEnv("JWT_SECRET", "default-secret"),
		JWTExpireHours: getEnv("JWT_EXPIRE_HOURS", "24"),
		FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
