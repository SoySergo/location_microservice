package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Cache    CacheConfig
	Log      LogConfig
	Worker   WorkerConfig
}

type ServerConfig struct {
	Host string
	Port int
	Env  string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type CacheConfig struct {
	TilesCacheTTL  time.Duration
	SearchCacheTTL time.Duration
}

type LogConfig struct {
	Level string
}

type WorkerConfig struct {
	Enabled           bool
	ConsumerGroup     string
	StreamReadTimeout time.Duration
	MaxRetries        int
	TransportRadius   float64
	TransportTypes    []string
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("API_HOST"),
			Port: viper.GetInt("API_PORT"),
			Env:  viper.GetString("API_ENV"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("DB_HOST"),
			Port:            viper.GetInt("DB_PORT"),
			User:            viper.GetString("DB_USER"),
			Password:        viper.GetString("DB_PASSWORD"),
			DBName:          viper.GetString("DB_NAME"),
			SSLMode:         viper.GetString("DB_SSLMODE"),
			MaxConns:        viper.GetInt("DB_MAX_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME")) * time.Second,
			ConnMaxIdleTime: time.Duration(viper.GetInt("DB_CONN_MAX_IDLE_TIME")) * time.Second,
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetInt("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		Cache: CacheConfig{
			TilesCacheTTL:  time.Duration(viper.GetInt("TILES_CACHE_TTL")) * time.Second,
			SearchCacheTTL: time.Duration(viper.GetInt("SEARCH_CACHE_TTL")) * time.Second,
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		Worker: WorkerConfig{
			Enabled:           viper.GetBool("WORKER_ENABLED"),
			ConsumerGroup:     viper.GetString("WORKER_CONSUMER_GROUP"),
			StreamReadTimeout: time.Duration(viper.GetInt("WORKER_STREAM_READ_TIMEOUT")) * time.Millisecond,
			MaxRetries:        viper.GetInt("WORKER_MAX_RETRIES"),
			TransportRadius:   viper.GetFloat64("WORKER_TRANSPORT_RADIUS"),
			TransportTypes:    parseTransportTypes(viper.GetString("WORKER_TRANSPORT_TYPES")),
		},
	}

	// Set default values if not provided
	if cfg.Worker.ConsumerGroup == "" {
		cfg.Worker.ConsumerGroup = "location-enrichment-workers"
	}
	if cfg.Worker.StreamReadTimeout == 0 {
		cfg.Worker.StreamReadTimeout = 5000 * time.Millisecond
	}
	if cfg.Worker.MaxRetries == 0 {
		cfg.Worker.MaxRetries = 3
	}
	if cfg.Worker.TransportRadius == 0 {
		cfg.Worker.TransportRadius = 1000
	}
	if len(cfg.Worker.TransportTypes) == 0 {
		cfg.Worker.TransportTypes = []string{"metro", "train", "tram", "bus"}
	}

	return cfg, nil
}

func parseTransportTypes(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
