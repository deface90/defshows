// Package config loads defShows service configuration from the environment.
//
// It is intentionally dependency-free (plain os.Getenv) so every service can
// load only the sections it needs. New sections are added as features land.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the aggregate configuration. Services embed/consume the sections
// they need.
type Config struct {
	DB     DB
	Log    Log
	Server Server
	JWT      JWT
	TMDB     TMDB
	OMDb     OMDb
	Worker   Worker
	Telegram Telegram
	Notifier Notifier
	OAuth    OAuth
	Seed     Seed
}

// Seed holds optional bootstrap data applied on startup after migrations.
type Seed struct {
	AdminEmail    string
	AdminPassword string
}

// OAuth holds social-login settings.
type OAuth struct {
	FrontendURL     string
	RedirectBaseURL string
	Google          OAuthProviderConfig
	Yandex          OAuthProviderConfig
	VK              OAuthProviderConfig
}

// OAuthProviderConfig is a single provider's client credentials.
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
}

// Enabled reports whether the provider is configured.
func (c OAuthProviderConfig) Enabled() bool { return c.ClientID != "" && c.ClientSecret != "" }

// Telegram holds bot settings.
type Telegram struct {
	Token    string
	Username string
	BaseURL  string
}

// Notifier holds notification worker settings.
type Notifier struct {
	ScanInterval time.Duration
	SendInterval time.Duration
	Lookback     time.Duration
	LinkTTL      time.Duration
}

// OMDb holds ratings provider settings.
type OMDb struct {
	APIKey string
}

// Worker holds background sync settings.
type Worker struct {
	SyncInterval time.Duration
	StaleAge     time.Duration
	Throttle     time.Duration
	Batch        int
}

// TMDB holds catalog provider settings.
type TMDB struct {
	APIKey   string
	Language string
}

// Server holds HTTP server settings.
type Server struct {
	Addr string
	// CORSAllowedOrigins are the browser origins allowed to call the API.
	CORSAllowedOrigins []string
}

// JWT holds token settings. Secret is required by services that issue/verify
// tokens; they check it at startup.
type JWT struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// DB holds Postgres connection settings.
type DB struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Log holds logging settings.
type Log struct {
	Level  string // debug|info|warn|error
	Format string // json|text
}

// Load reads configuration from the environment, applying defaults, and
// validates it.
func Load() (Config, error) {
	cfg := Config{
		DB: DB{
			DSN:             getEnv("DB_DSN", ""),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 10),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", time.Hour),
		},
		Log: Log{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Server: Server{
			Addr:               getEnv("HTTP_ADDR", ":8080"),
			CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:5173"}),
		},
		JWT: JWT{
			Secret:     getEnv("JWT_SECRET", ""),
			AccessTTL:  getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTTL: getEnvDuration("REFRESH_TOKEN_TTL", 720*time.Hour),
		},
		TMDB: TMDB{
			APIKey:   getEnv("TMDB_API_KEY", ""),
			Language: getEnv("TMDB_LANGUAGE", "ru-RU"),
		},
		OMDb: OMDb{
			APIKey: getEnv("OMDB_API_KEY", ""),
		},
		Worker: Worker{
			SyncInterval: getEnvDuration("WORKER_SYNC_INTERVAL", time.Hour),
			StaleAge:     getEnvDuration("WORKER_STALE_AGE", 24*time.Hour),
			Throttle:     getEnvDuration("WORKER_THROTTLE", 250*time.Millisecond),
			Batch:        getEnvInt("WORKER_BATCH", 100),
		},
		Telegram: Telegram{
			Token:    getEnv("TELEGRAM_BOT_TOKEN", ""),
			Username: getEnv("TELEGRAM_BOT_USERNAME", ""),
			BaseURL:  getEnv("TELEGRAM_API_BASE_URL", "https://api.telegram.org"),
		},
		Notifier: Notifier{
			ScanInterval: getEnvDuration("NOTIFIER_SCAN_INTERVAL", 15*time.Minute),
			SendInterval: getEnvDuration("NOTIFIER_SEND_INTERVAL", time.Minute),
			Lookback:     getEnvDuration("NOTIFIER_LOOKBACK", 7*24*time.Hour),
			LinkTTL:      getEnvDuration("NOTIFIER_LINK_TTL", 15*time.Minute),
		},
		OAuth: OAuth{
			FrontendURL:     getEnv("FRONTEND_URL", "http://localhost:8080"),
			RedirectBaseURL: getEnv("OAUTH_REDIRECT_BASE_URL", "http://localhost:8081"),
			Google:          OAuthProviderConfig{ClientID: getEnv("OAUTH_GOOGLE_CLIENT_ID", ""), ClientSecret: getEnv("OAUTH_GOOGLE_CLIENT_SECRET", "")},
			Yandex:          OAuthProviderConfig{ClientID: getEnv("OAUTH_YANDEX_CLIENT_ID", ""), ClientSecret: getEnv("OAUTH_YANDEX_CLIENT_SECRET", "")},
			VK:              OAuthProviderConfig{ClientID: getEnv("OAUTH_VK_CLIENT_ID", ""), ClientSecret: getEnv("OAUTH_VK_CLIENT_SECRET", "")},
		},
		Seed: Seed{
			AdminEmail:    getEnv("SEED_ADMIN_EMAIL", ""),
			AdminPassword: getEnv("SEED_ADMIN_PASSWORD", ""),
		},
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate reports configuration errors.
func (c Config) Validate() error {
	if c.DB.DSN == "" {
		return fmt.Errorf("config: DB_DSN is required")
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("config: invalid LOG_LEVEL %q", c.Log.Level)
	}
	switch c.Log.Format {
	case "json", "text":
	default:
		return fmt.Errorf("config: invalid LOG_FORMAT %q", c.Log.Format)
	}
	return nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// getEnvList reads a comma-separated env var into a trimmed, non-empty slice.
func getEnvList(key string, def []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}

func getEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
