package config

import (
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	PostgresURL  string
	ServerAddr   string
}

// Load загружает конфигурацию из переменных окружения.
func Load() *Config {
	return &Config{
		KafkaBrokers: getEnvAsSlice("KAFKA_BROKERS", "localhost:9092,localhost:9094,localhost:9096"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "orders"),
		PostgresURL:  getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/postgres_db"),
		ServerAddr:   getEnv("SERVER_ADDR", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsSlice(name, fallback string) []string {
	val := getEnv(name, fallback)
	return strings.Split(val, ",")
}
