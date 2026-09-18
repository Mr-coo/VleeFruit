// Package config loads runtime configuration from environment variables.
// It reads a .env file first (if present) so local runs match Compose.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config is the fully-resolved application configuration.
type Config struct {
	HTTPPort   string
	AdminToken string

	// Env is the deployment environment (development|production); it selects the
	// log format and Gin's mode. LogLevel is debug|info|warn|error.
	Env      string
	LogLevel string

	HTTP     HTTPConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	S3       S3Config
	MQTT     MQTTConfig
}

// HTTPConfig tunes the request-facing protections on the REST API.
type HTTPConfig struct {
	// MaxBodyBytes caps the request body the admin API will read (0 = unlimited).
	MaxBodyBytes int64
	// RateLimitRPS is the sustained per-client request rate (0 = disabled).
	RateLimitRPS float64
	// RateLimitBurst is the bucket size allowing short bursts above the rate.
	RateLimitBurst int
}

type PostgresConfig struct {
	// DSN, when set, wins over the individual fields below.
	DSN      string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type S3Config struct {
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	Bucket         string
	UseSSL         bool
	ForcePathStyle bool
}

type MQTTConfig struct {
	BrokerURL string // e.g. tls://mosquitto:8883 or tcp://mosquitto:1883
	ClientID  string
	Username  string
	Password  string
	CAFile    string // path to the broker CA cert for TLS verification
	Insecure  bool   // skip TLS verification (dev only)

	// MaxImageBytes caps the raw image payload accepted from a device over MQTT
	// before decoding (0 = unlimited). Guards against memory-exhaustion DoS.
	MaxImageBytes int64
}

// IsProduction reports whether the app runs in a production-like environment.
func (c *Config) IsProduction() bool {
	switch strings.ToLower(strings.TrimSpace(c.Env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

// DSN returns a GORM-compatible Postgres connection string.
func (p PostgresConfig) buildDSN() string {
	if p.DSN != "" {
		return p.DSN
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Name, p.SSLMode)
}

// Load builds a Config from the environment. It never fails on a missing
// .env file; it only surfaces genuinely invalid values.
func Load() (*Config, error) {
	// Best-effort: local dev has a .env, containers inject real env vars.
	_ = godotenv.Load()

	pg := PostgresConfig{
		DSN:      os.Getenv("DATABASE_URL"),
		Host:     env("DB_HOST", "localhost"),
		Port:     env("DB_PORT", "5432"),
		User:     env("DB_USER", "vleefruit"),
		Password: env("DB_PASSWORD", "vleefruit"),
		Name:     env("DB_NAME", "vleefruit"),
		SSLMode:  env("DB_SSLMODE", "disable"),
	}
	// Normalise so callers only ever read DSN.
	pg.DSN = pg.buildDSN()

	cfg := &Config{
		HTTPPort:   env("PORT", "8080"),
		AdminToken: env("ADMIN_TOKEN", "dev-admin-token"),
		Env:        env("APP_ENV", "development"),
		LogLevel:   env("LOG_LEVEL", "info"),
		HTTP: HTTPConfig{
			MaxBodyBytes:   envInt64("HTTP_MAX_BODY_BYTES", 1<<20), // 1 MiB
			RateLimitRPS:   envFloat("RATE_LIMIT_RPS", 10),
			RateLimitBurst: envInt("RATE_LIMIT_BURST", 20),
		},
		Postgres: pg,
		Redis: RedisConfig{
			Addr:     env("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       envInt("REDIS_DB", 0),
		},
		S3: S3Config{
			Endpoint:       env("S3_ENDPOINT", "http://localhost:9000"),
			Region:         env("S3_REGION", "us-east-1"),
			AccessKey:      env("S3_ACCESS_KEY", "minioadmin"),
			SecretKey:      env("S3_SECRET_KEY", "minioadmin"),
			Bucket:         env("S3_BUCKET", "fruit-images"),
			UseSSL:         envBool("S3_USE_SSL", false),
			ForcePathStyle: envBool("S3_FORCE_PATH_STYLE", true),
		},
		MQTT: MQTTConfig{
			BrokerURL: env("MQTT_BROKER_URL", "tcp://localhost:1883"),
			ClientID:  env("MQTT_CLIENT_ID", "vleefruit-backend"),
			Username:  env("MQTT_USERNAME", "backend"),
			Password:  os.Getenv("MQTT_PASSWORD"),
			CAFile:        os.Getenv("MQTT_TLS_CA"),
			Insecure:      envBool("MQTT_TLS_INSECURE", false),
			MaxImageBytes: envInt64("MQTT_MAX_IMAGE_BYTES", 8<<20), // 8 MiB
		},
	}

	return cfg, nil
}
