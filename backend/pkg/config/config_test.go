package config_test

import (
	"testing"

	"github.com/deface90/defshows/backend/pkg/config"
)

func TestLoad(t *testing.T) {
	// Keys the loader reads; reset before each case for isolation.
	keys := []string{
		"DB_DSN", "LOG_LEVEL", "LOG_FORMAT",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME",
	}

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
		check   func(t *testing.T, c config.Config)
	}{
		{
			name: "defaults with dsn",
			env:  map[string]string{"DB_DSN": "postgres://x"},
			check: func(t *testing.T, c config.Config) {
				if c.Log.Level != "info" || c.Log.Format != "json" {
					t.Fatalf("unexpected log defaults: %+v", c.Log)
				}
				if c.DB.MaxOpenConns != 10 || c.DB.MaxIdleConns != 5 {
					t.Fatalf("unexpected pool defaults: %+v", c.DB)
				}
				if c.DB.ConnMaxLifetime.Hours() != 1 {
					t.Fatalf("want 1h lifetime, got %v", c.DB.ConnMaxLifetime)
				}
			},
		},
		{
			name:    "missing dsn",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name:    "invalid log level",
			env:     map[string]string{"DB_DSN": "x", "LOG_LEVEL": "bogus"},
			wantErr: true,
		},
		{
			name:    "invalid log format",
			env:     map[string]string{"DB_DSN": "x", "LOG_FORMAT": "xml"},
			wantErr: true,
		},
		{
			name: "override ints and duration",
			env: map[string]string{
				"DB_DSN":               "x",
				"DB_MAX_OPEN_CONNS":    "42",
				"DB_CONN_MAX_LIFETIME": "30m",
				"LOG_LEVEL":            "debug",
				"LOG_FORMAT":           "text",
			},
			check: func(t *testing.T, c config.Config) {
				if c.DB.MaxOpenConns != 42 {
					t.Fatalf("want MaxOpenConns=42, got %d", c.DB.MaxOpenConns)
				}
				if c.DB.ConnMaxLifetime.Minutes() != 30 {
					t.Fatalf("want 30m, got %v", c.DB.ConnMaxLifetime)
				}
				if c.Log.Level != "debug" || c.Log.Format != "text" {
					t.Fatalf("unexpected log: %+v", c.Log)
				}
			},
		},
		{
			name: "invalid int falls back to default",
			env:  map[string]string{"DB_DSN": "x", "DB_MAX_OPEN_CONNS": "notanint"},
			check: func(t *testing.T, c config.Config) {
				if c.DB.MaxOpenConns != 10 {
					t.Fatalf("want fallback 10, got %d", c.DB.MaxOpenConns)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range keys {
				t.Setenv(k, "") // treated as unset by the loader
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			c, err := config.Load()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, c)
			}
		})
	}
}
