package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Redis    RedisConfig
	AWS      AWSConfig
	Services ServicesConfig
	Worker   WorkerConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type AWSConfig struct {
	Endpoint                string
	Region                  string
	SQSQueueURL             string
	DynamoDBConversionsTable string
	DynamoDBLogsTable       string
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
	Concurrency int
	JobTimeout  time.Duration
}

func Load() *Config {
	return &Config{
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		AWS: AWSConfig{
			Endpoint:                 getEnv("AWS_ENDPOINT", ""),
			Region:                   getEnv("AWS_REGION", "us-east-1"),
			SQSQueueURL:              getEnv("SQS_QUEUE_URL", ""),
			DynamoDBConversionsTable: getEnv("DYNAMODB_CONVERSIONS_TABLE", "playswap-conversions"),
			DynamoDBLogsTable:        getEnv("DYNAMODB_LOGS_TABLE", "playswap-conversion-logs"),
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
			Concurrency: getEnvInt("WORKER_CONCURRENCY", 5),
			JobTimeout:  getEnvDuration("WORKER_JOB_TIMEOUT", 5*time.Minute),
		},
	}
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
