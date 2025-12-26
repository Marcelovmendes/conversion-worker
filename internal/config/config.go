package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Redis    RedisConfig
	Postgres PostgresConfig
	Services ServicesConfig
	Worker   WorkerConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

type ServicesConfig struct {
	Spotify ServiceConfig
	YouTube ServiceConfig
}

type ServiceConfig struct {
	BaseURL string
	Timeout time.Duration
}

type WorkerConfig struct {
	Concurrency  int
	PollInterval time.Duration
	JobTimeout   time.Duration
}

func Load() *Config {
	return &Config{
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			Database: getEnv("POSTGRES_DATABASE", "playswap"),
			User:     getEnv("POSTGRES_USER", "marcelomendes"),
			Password: getEnv("POSTGRES_PASSWORD", "developer_user"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Services: ServicesConfig{
			Spotify: ServiceConfig{
				BaseURL: getEnv("SPOTIFY_SERVICE_URL", "http://localhost:8080"),
				Timeout: getEnvDuration("SPOTIFY_SERVICE_TIMEOUT", 30*time.Second),
			},
			YouTube: ServiceConfig{
				BaseURL: getEnv("YOUTUBE_SERVICE_URL", "http://localhost:8081"),
				Timeout: getEnvDuration("YOUTUBE_SERVICE_TIMEOUT", 30*time.Second),
			},
		},
		Worker: WorkerConfig{
			Concurrency:  getEnvInt("WORKER_CONCURRENCY", 5),
			PollInterval: getEnvDuration("WORKER_POLL_INTERVAL", 1*time.Second),
			JobTimeout:   getEnvDuration("WORKER_JOB_TIMEOUT", 5*time.Minute),
		},
	}
}

func (c *PostgresConfig) ConnectionString() string {
	return "postgres://" + c.User + ":" + c.Password + "@" + c.Host + ":" + strconv.Itoa(c.Port) + "/" + c.Database + "?sslmode=" + c.SSLMode
}

func (c *RedisConfig) Address() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
