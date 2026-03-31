package config_test

import (
	"os"
	"testing"
	"time"

	"clippy/internal/config"
)

func TestLoad_RequiresSecret(t *testing.T) {
	os.Unsetenv("CLIPPY_SECRET")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when CLIPPY_SECRET is missing")
	}
}

func TestLoad_SecretTooShort(t *testing.T) {
	os.Setenv("CLIPPY_SECRET", "short")
	defer os.Unsetenv("CLIPPY_SECRET")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when CLIPPY_SECRET is too short")
	}
}

func TestLoad_Defaults(t *testing.T) {
	os.Setenv("CLIPPY_SECRET", "a-secret-value-that-is-long-enough-yes!!")
	defer os.Unsetenv("CLIPPY_SECRET")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %q", cfg.Port)
	}
	if cfg.DefaultTTLDuration != time.Hour*24 {
		t.Errorf("expected default TTL 24, got %d", cfg.DefaultTTLDuration)
	}
	if cfg.DBPath != "/data/clippy.db" {
		t.Errorf("expected default db path /data/clippy.db, got %q", cfg.DBPath)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	os.Setenv("CLIPPY_SECRET", "a-secret-value-that-is-long-enough-yes!!")
	os.Setenv("CLIPPY_PORT", "9090")
	os.Setenv("CLIPPY_DEFAULT_TTL_HOURS", "6")
	os.Setenv("CLIPPY_DB_PATH", "/tmp/test.db")
	defer func() {
		os.Unsetenv("CLIPPY_SECRET")
		os.Unsetenv("CLIPPY_PORT")
		os.Unsetenv("CLIPPY_DEFAULT_TTL_HOURS")
		os.Unsetenv("CLIPPY_DB_PATH")
	}()

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %q", cfg.Port)
	}
	if cfg.DefaultTTLDuration != time.Hour*6 {
		t.Errorf("expected TTL 6, got %d", cfg.DefaultTTLDuration)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %q", cfg.DBPath)
	}
}

func TestLoad_InvalidTTL(t *testing.T) {
	tc := []struct {
		name string
		ttl  string
	}{
		{
			name: "not a number",
			ttl:  "nonumber",
		},
		{
			name: "zero hours",
			ttl:  "0",
		},
		{
			name: "negative hours",
			ttl:  "-8",
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("CLIPPY_SECRET", "a-secret-value-that-is-long-enough-yes!!")
			os.Setenv("CLIPPY_DEFAULT_TTL_HOURS", tc.ttl)
			defer func() {
				os.Unsetenv("CLIPPY_SECRET")
				os.Unsetenv("CLIPPY_DEFAULT_TTL_HOURS")
			}()

			_, err := config.Load()
			if err == nil {
				t.Fatal("expected error for invalid TTL value")
			}
		})
	}
}
