package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

// AppConfig holds runtime configuration flags.
type AppConfig struct {
	AppName         string
	Environment     string
	HTTPPort        string
	GRPCPort        string
	DatabaseDriver  string
	DatabaseDSN     string
	JWTSecret       string
	RefreshSecret   string
	TokenTTL        int64
	RefreshTokenTTL int64
	OssEndpoint     string
	OssAccessKey    string
	OssSecretKey    string
	OssBucket       string
}

var (
	cfg  AppConfig
	once sync.Once
)

// Load reads configuration from environment variables. It is safe for concurrent use.
func Load() AppConfig {
	once.Do(func() {
		loadDotEnv()

		cfg = AppConfig{
			AppName:         getEnv("APP_NAME", "LearnGo"),
			Environment:     getEnv("APP_ENV", "local"),
			HTTPPort:        getEnv("HTTP_PORT", "8080"),
			GRPCPort:        getEnv("GRPC_PORT", "9090"),
			DatabaseDriver:  getEnv("DATABASE_DRIVER", "sqlite"),
			DatabaseDSN:     getEnv("DATABASE_DSN", "file:learn-go.db?cache=shared&_foreign_keys=on"),
			JWTSecret:       mustEnv("JWT_SECRET"),
			RefreshSecret:   mustEnv("REFRESH_SECRET"),
			TokenTTL:        getEnvAsInt64("TOKEN_TTL", 3600),
			RefreshTokenTTL: getEnvAsInt64("REFRESH_TOKEN_TTL", 2592000),
			OssEndpoint:     getEnv("OSS_ENDPOINT", ""),
			OssAccessKey:    getEnv("OSS_ACCESS_KEY", ""),
			OssSecretKey:    getEnv("OSS_SECRET_KEY", ""),
			OssBucket:       getEnv("OSS_BUCKET", ""),
		}
	})

	return cfg
}

func loadDotEnv() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("config: unable to determine working directory: %v", err)
		return
	}

	for _, candidate := range []string{".env", filepath.Join(cwd, "config", ".env")} {
		if _, err := os.Stat(candidate); err == nil {
			if err := godotenv.Load(candidate); err != nil {
				log.Printf("config: unable to load %s: %v", candidate, err)
			}
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		var parsed int64
		_, err := fmt.Sscan(value, &parsed)
		if err == nil {
			return parsed
		}
		log.Printf("config: unable to parse %s=%s as int64: %v", key, value, err)
	}
	return fallback
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("config: missing required environment variable %s", key)
	}
	return value
}
