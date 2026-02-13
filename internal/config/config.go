package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Neo4j    Neo4jConfig
	Bedrock  BedrockConfig
	Valkey   ValkeyConfig
	MinIO    MinIOConfig
	S3       S3Config
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

type Neo4jConfig struct {
	URI      string
	User     string
	Password string
}

type BedrockConfig struct {
	Region  string
	ModelID string
}

type ValkeyConfig struct {
	Addr     string
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type S3Config struct {
	Region   string // S3_REGION
	Bucket   string // S3_BUCKET
	Prefix   string // S3_PREFIX (optional default prefix)
	Endpoint string // S3_ENDPOINT (for MinIO/LocalStack compatibility)
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Port:         getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT_SECS", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT_SECS", 60)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "codegraph"),
			Password: getEnv("DB_PASSWORD", "codegraph"),
			Name:     getEnv("DB_NAME", "codegraph"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			MaxConns: int32(getEnvInt("DB_MAX_CONNS", 25)),
			MinConns: int32(getEnvInt("DB_MIN_CONNS", 5)),
		},
		Neo4j: Neo4jConfig{
			URI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
			User:     getEnv("NEO4J_USER", "neo4j"),
			Password: getEnv("NEO4J_PASSWORD", "codegraph"),
		},
		Bedrock: BedrockConfig{
			Region:  getEnv("BEDROCK_REGION", "us-east-1"),
			ModelID: getEnv("BEDROCK_MODEL_ID", "cohere.embed-english-v4"),
		},
		Valkey: ValkeyConfig{
			Addr:     getEnv("VALKEY_ADDR", "localhost:6379"),
			Password: getEnv("VALKEY_PASSWORD", ""),
			DB:       getEnvInt("VALKEY_DB", 0),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "codegraph"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "codegraph123"),
			Bucket:    getEnv("MINIO_BUCKET", "codegraph"),
			UseSSL:    getEnvBool("MINIO_USE_SSL", false),
		},
		S3: S3Config{
			Region:   getEnv("S3_REGION", ""),
			Bucket:   getEnv("S3_BUCKET", ""),
			Prefix:   getEnv("S3_PREFIX", ""),
			Endpoint: getEnv("S3_ENDPOINT", ""),
		},
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
