package config

import (
	"os"
)

type Config struct {
	AppEnv   string
	HTTPPort string
	LogLevel string
}

func Load() Config {
	return Config{
		AppEnv:   getenv("APP_ENV", "local"),
		HTTPPort: getenv("HTTP_PORT", "8080"),
		LogLevel: getenv("LOG_LEVEL", "info"),
	}
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
