package config

import (
	"log/slog"
	"os"
)

func Load() App {
	cfg := App{
		Port:         getenv("APP_PORT", "8080"),
		DatabaseURL:  must("DATABASE_URL"),
		JWTSecret:    getenv("JWT_SECRET", "local_dev_secret"),
		ApiNinjasKey: os.Getenv("API_NINJAS_KEY"),
		Env:          getenv("APP_ENV", "dev"),
	}
	return cfg
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(k string) string {
	v := os.Getenv(k)
	if v == "" {
		slog.Error("required env missing", "key", k)
		panic("missing env " + k)
	}
	return v
}
