package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	defaultTTL    = time.Hour * 24
	defaultPort   = "8080"
	defaultDBPath = "/data/clippy.db"
)

type Config struct {
	Port               string
	DefaultTTLDuration time.Duration
	DBPath             string
	Secret             string
}

func getEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return v, fmt.Errorf("%q not set", key)
	}
	return v, nil
}

func Load() (*Config, error) {
	cfg := &Config{}

	secret, err := getEnv("CLIPPY_SECRET")
	if err != nil {
		return nil, err
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("CLIPPY_SECRET must be at least 32 characters long")
	}
	cfg.Secret = secret

	port, err := getEnv("CLIPPY_PORT")
	if err != nil {
		cfg.Port = defaultPort
	} else {
		cfg.Port = port
	}

	ttlHours, err := getEnv("CLIPPY_DEFAULT_TTL_HOURS")
	if err == nil {
		ttlHoursInt, err := strconv.Atoi(ttlHours)
		if err != nil {
			return nil, fmt.Errorf("converting CLIPPY_DEFAULT_TTL_HOURS to int: %w", err)
		}
		if ttlHoursInt <= 0 {
			return nil, fmt.Errorf("CLIPPY_DEFAULT_TTL_HOURS must be positive nonzero: %d", ttlHoursInt)
		}
		cfg.DefaultTTLDuration = time.Duration(ttlHoursInt) * time.Hour
	} else {
		cfg.DefaultTTLDuration = defaultTTL
	}

	dbPath, err := getEnv("CLIPPY_DB_PATH")
	if err != nil {
		cfg.DBPath = defaultDBPath
	} else {
		cfg.DBPath = dbPath
	}

	return cfg, nil
}
