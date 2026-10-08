// Package config loads and validates environment configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env, Port, DatabaseURL                     string
	JWTSecret, RefreshSecret                   string
	FrontendURL, StaticDir, PublicURL          string
	StorageDriver, StoragePath                 string
	PaymentProvider, WebhookSecret             string
	SMTPHost, SMTPPort, SMTPUser, SMTPPassword string
	MailFrom                                   string
	AdminEmail, AdminPassword                  string
	AutoMigrate, AutoSeed, TrustProxy          bool
	AuthRate, AuthBurst, APIRate, APIBurst     int
}

func get(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
func geti(k string, d int) int {
	if n, err := strconv.Atoi(os.Getenv(k)); err == nil && n > 0 {
		return n
	}
	return d
}
func flag(k string, d bool) bool {
	v := strings.ToLower(get(k, ""))
	if v == "" {
		return d
	}
	return v == "1" || v == "true" || v == "yes"
}

func Load() (Config, error) {
	c := Config{
		Env: get("APP_ENV", "development"), Port: get("APP_PORT", "8080"), DatabaseURL: get("DATABASE_URL", ""),
		JWTSecret: get("JWT_SECRET", "dev-only-insecure-jwt-secret-change-me"), RefreshSecret: get("JWT_REFRESH_SECRET", "dev-only-insecure-refresh-secret-change-me"),
		FrontendURL: get("FRONTEND_URL", "http://localhost:5500"), StaticDir: get("STATIC_DIR", ""), PublicURL: get("PUBLIC_URL", ""),
		StorageDriver: get("STORAGE_DRIVER", "local"), StoragePath: get("STORAGE_PATH", "./uploads"),
		PaymentProvider: get("PAYMENT_PROVIDER", "test"), WebhookSecret: get("PAYMENT_WEBHOOK_SECRET", "dev-only-webhook-secret-change-me"),
		SMTPHost: get("SMTP_HOST", ""), SMTPPort: get("SMTP_PORT", "587"), SMTPUser: get("SMTP_USER", ""), SMTPPassword: get("SMTP_PASSWORD", ""),
		MailFrom: get("MAIL_FROM", "no-reply@localhost"), AdminEmail: get("ADMIN_EMAIL", ""), AdminPassword: get("ADMIN_PASSWORD", ""),
		AutoMigrate: flag("AUTO_MIGRATE", true), AutoSeed: flag("AUTO_SEED", true), TrustProxy: flag("TRUST_PROXY", false),
		AuthRate: geti("RATE_AUTH_PER_MIN", 20), AuthBurst: geti("RATE_AUTH_BURST", 10), APIRate: geti("RATE_API_PER_MIN", 600), APIBurst: geti("RATE_API_BURST", 200),
	}
	if c.PublicURL == "" {
		c.PublicURL = "http://localhost:" + c.Port
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.IsProd() {
		for k, v := range map[string]string{"JWT_SECRET": c.JWTSecret, "JWT_REFRESH_SECRET": c.RefreshSecret, "PAYMENT_WEBHOOK_SECRET": c.WebhookSecret} {
			if len(v) < 32 || strings.Contains(v, "change") || strings.Contains(v, "dev-only") {
				return c, fmt.Errorf("%s must be a random value of at least 32 characters in production", k)
			}
		}
		if c.PaymentProvider == "test" {
			return c, fmt.Errorf("PAYMENT_PROVIDER=test is not allowed in production")
		}
	}
	return c, nil
}

func (c Config) IsProd() bool { return c.Env == "production" }

// Origins returns the allowed browser origins (FRONTEND_URL may be comma separated).
func (c Config) Origins() []string {
	var o []string
	for _, s := range strings.Split(c.FrontendURL, ",") {
		if s = strings.TrimRight(strings.TrimSpace(s), "/"); s != "" {
			o = append(o, s)
		}
	}
	return o
}

// CookieSecure reports whether auth cookies must carry the Secure flag.
func (c Config) CookieSecure() bool { return c.IsProd() }
