package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "configs/config.yaml"

type Config struct {
	Server          ServerConfig    `yaml:"server"`
	Database        DatabaseConfig  `yaml:"database"`
	ParkingDatabase DatabaseConfig  `yaml:"parking_database"`
	Redis           RedisConfig     `yaml:"redis"`
	Auth            AuthConfig      `yaml:"auth"`
	Telemetry       TelemetryConfig `yaml:"telemetry"`
}

type AuthConfig struct {
	Issuer         string `yaml:"issuer"`
	AccessTTL      string `yaml:"access_ttl"`
	RefreshTTL     string `yaml:"refresh_ttl"`
	PrivateKeyPath string `yaml:"private_key_path"`
	PublicKeyPath  string `yaml:"public_key_path"`
	KeyID          string `yaml:"key_id"`
}

type ServerConfig struct {
	GRPCPort       int `yaml:"grpc_port"`
	MetricsPort    int `yaml:"metrics_port"`
	HTTPHealthPort int `yaml:"http_health_port"`
}

type DatabaseConfig struct {
	URL      string `yaml:"url"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type TelemetryConfig struct {
	ServiceName string        `yaml:"service_name"`
	Environment string        `yaml:"environment"`
	Log         LogConfig     `yaml:"log"`
	Trace       TraceConfig   `yaml:"trace"`
	Metrics     MetricsConfig `yaml:"metrics"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type TraceConfig struct {
	Endpoint      string  `yaml:"endpoint"`
	Insecure      bool    `yaml:"insecure"`
	SamplingRatio float64 `yaml:"sampling_ratio"`
}

type MetricsConfig struct {
	Path string `yaml:"path"`
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			GRPCPort:       50051,
			MetricsPort:    9090,
			HTTPHealthPort: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "parkhub",
			Password: "parkhub",
			DBName:   "parkhub_identity",
		},
		ParkingDatabase: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "parkhub",
			Password: "parkhub",
			DBName:   "parkhub_parking",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		Auth: AuthConfig{
			Issuer:         "parkhub",
			AccessTTL:      "15m",
			RefreshTTL:     "168h",
			PrivateKeyPath: "configs/keys/jwt_private.pem",
			PublicKeyPath:  "configs/keys/jwt_public.pem",
			KeyID:          "parkhub-2026-04",
		},
		Telemetry: TelemetryConfig{
			ServiceName: "parkhub-monolith",
			Environment: "development",
			Log: LogConfig{
				Level:  "info",
				Format: "json",
			},
			Trace: TraceConfig{
				Endpoint:      "localhost:4317",
				Insecure:      true,
				SamplingRatio: 1,
			},
			Metrics: MetricsConfig{
				Path: "/metrics",
			},
		},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}

	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return Config{}, fmt.Errorf("read config file: %w", err)
		}
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config file: %w", err)
	}

	applyEnvOverrides(&cfg)
	return cfg, nil
}

func (c DatabaseConfig) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
	)
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("GRPC_PORT"); v != "" {
		cfg.Server.GRPCPort = mustInt(v, cfg.Server.GRPCPort)
	}
	if v := os.Getenv("METRICS_PORT"); v != "" {
		cfg.Server.MetricsPort = mustInt(v, cfg.Server.MetricsPort)
	}
	if v := os.Getenv("HTTP_HEALTH_PORT"); v != "" {
		cfg.Server.HTTPHealthPort = mustInt(v, cfg.Server.HTTPHealthPort)
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		cfg.Database.Port = mustInt(v, cfg.Database.Port)
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("PARKING_DATABASE_URL"); v != "" {
		cfg.ParkingDatabase.URL = v
	}
	if v := os.Getenv("PARKING_DB_HOST"); v != "" {
		cfg.ParkingDatabase.Host = v
	}
	if v := os.Getenv("PARKING_DB_PORT"); v != "" {
		cfg.ParkingDatabase.Port = mustInt(v, cfg.ParkingDatabase.Port)
	}
	if v := os.Getenv("PARKING_DB_USER"); v != "" {
		cfg.ParkingDatabase.User = v
	}
	if v := os.Getenv("PARKING_DB_PASSWORD"); v != "" {
		cfg.ParkingDatabase.Password = v
	}
	if v := os.Getenv("PARKING_DB_NAME"); v != "" {
		cfg.ParkingDatabase.DBName = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		cfg.Redis.DB = mustInt(v, cfg.Redis.DB)
	}
	if v := os.Getenv("AUTH_ISSUER"); v != "" {
		cfg.Auth.Issuer = v
	}
	if v := os.Getenv("AUTH_ACCESS_TTL"); v != "" {
		cfg.Auth.AccessTTL = v
	}
	if v := os.Getenv("AUTH_REFRESH_TTL"); v != "" {
		cfg.Auth.RefreshTTL = v
	}
	if v := os.Getenv("AUTH_PRIVATE_KEY_PATH"); v != "" {
		cfg.Auth.PrivateKeyPath = v
	}
	if v := os.Getenv("AUTH_PUBLIC_KEY_PATH"); v != "" {
		cfg.Auth.PublicKeyPath = v
	}
	if v := os.Getenv("AUTH_KEY_ID"); v != "" {
		cfg.Auth.KeyID = v
	}
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.Telemetry.ServiceName = v
	}
	if v := os.Getenv("OTEL_ENVIRONMENT"); v != "" {
		cfg.Telemetry.Environment = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Telemetry.Log.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Telemetry.Log.Format = v
	}
	if v := os.Getenv("OTEL_TRACE_ENDPOINT"); v != "" {
		cfg.Telemetry.Trace.Endpoint = v
	}
	if v := os.Getenv("OTEL_TRACE_INSECURE"); v != "" {
		cfg.Telemetry.Trace.Insecure = mustBool(v, cfg.Telemetry.Trace.Insecure)
	}
	if v := os.Getenv("OTEL_TRACE_SAMPLING_RATIO"); v != "" {
		cfg.Telemetry.Trace.SamplingRatio = mustFloat(v, cfg.Telemetry.Trace.SamplingRatio)
	}
	if v := os.Getenv("METRICS_PATH"); v != "" {
		cfg.Telemetry.Metrics.Path = v
	}
}

func mustInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustBool(value string, fallback bool) bool {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
