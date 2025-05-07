package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type (
	// Config holds all the configuration settings
	Config struct {
		MinIO    *MinIOConfig
		Redis    *RedisConfig
		PsqlCfg  *PsqlConfig
		GRPCPort string
	}

	PsqlConfig struct {
		Dsn string
	}

	// MinIOConfig holds MinIO settings
	MinIOConfig struct {
		Endpoint  string
		AccessKey string
		SecretKey string
		Bucket    string
		UrlExpiry int
	}

	// RedisConfig holds Redis settings
	RedisConfig struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
)

// LoadConfig loads configurations from environment variables or .env file
func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Using system environment variables.")
	}

	return &Config{
		MinIO: &MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "admin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "secretpass"),
			Bucket:    getEnv("MINIO_BUCKET", "mediumlike"),
			UrlExpiry: getEnvInt("MINIO_URL_EXPIRY", 3_600),
		},
		Redis: &RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		PsqlCfg: &PsqlConfig{
			Dsn: getEnv("DB_DSN", "host=postgres user=postgres password=secret dbname=article_service port=5432 sslmode=disable TimeZone=Asia/Tashkent"),
		},
		GRPCPort: getEnv("GRPC_PORT", "7878"),
	}
}

// getEnv retrieves environment variables with a fallback default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvInt retrieves an integer environment variable
func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intValue int
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err == nil {
			return intValue
		}
	}
	return fallback
}
