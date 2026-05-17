package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds application settings loaded from env or config file.
type Config struct {
	AppName                   string `mapstructure:"app_name"`
	AppEnv                    string `mapstructure:"app_env"`
	HTTPAddr                  string `mapstructure:"http_addr"`
	LogLevel                  string `mapstructure:"log_level"`
	ConfigFile                string `mapstructure:"config_file"`
	DatabaseURL               string `mapstructure:"database_url"`
	RedisAddr                 string `mapstructure:"redis_addr"`
	KafkaBrokers              string `mapstructure:"kafka_brokers"`
	GatewayJWTToken           string `mapstructure:"gateway_jwt_token"`
	GatewayTimeoutSeconds     int    `mapstructure:"gateway_timeout_seconds"`
	AuthJWTSecret             string `mapstructure:"auth_jwt_secret"`
	AuthAccessTokenTTLMinutes int    `mapstructure:"auth_access_token_ttl_minutes"`
	AuthRefreshTokenTTLHours  int    `mapstructure:"auth_refresh_token_ttl_hours"`
}

// Load reads configuration from .env, environment variables, and optional config file.
func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetDefault("app_name", "golang-microservices")
	v.SetDefault("app_env", "development")
	v.SetDefault("http_addr", ":8080")
	v.SetDefault("log_level", "info")
	v.SetDefault("database_url", "postgres://postgres:postgres@localhost:5432/golang_microservices?sslmode=disable")
	v.SetDefault("redis_addr", "localhost:6379")
	v.SetDefault("kafka_brokers", "localhost:9092")
	v.SetDefault("gateway_jwt_token", "dev-gateway-token")
	v.SetDefault("gateway_timeout_seconds", 8)
	v.SetDefault("auth_jwt_secret", "dev-auth-secret")
	v.SetDefault("auth_access_token_ttl_minutes", 60)
	v.SetDefault("auth_refresh_token_ttl_hours", 168)

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = v.GetString("config_file")
	}
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.ConfigFile == "" {
		cfg.ConfigFile = configFile
	}

	return cfg, nil
}

func loadDotEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			return fmt.Errorf("load .env: %w", err)
		}
	}
	return nil
}
