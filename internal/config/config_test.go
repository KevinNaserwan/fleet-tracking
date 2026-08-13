package config

import (
	"strings"
	"testing"
)

func TestParseAndRedact(t *testing.T) {
	values := map[string]string{
		"APP_ENV":                 "production",
		"SERVICE_NAME":            "fleet-api",
		"HTTP_ADDR":               "0.0.0.0:8080",
		"LOG_LEVEL":               "info",
		"DATABASE_URL":            "postgres://fleet:database-secret@postgres:5432/fleet",
		"REDIS_ADDR":              "redis:6379",
		"REDIS_PASSWORD":          "redis-secret",
		"RABBITMQ_URL":            "amqp://fleet:rabbit-secret@rabbitmq:5672/",
		"JWT_ISSUER":              "fleet-tracking",
		"JWT_AUDIENCE":            "fleet-dashboard",
		"JWT_ACCESS_SECRET":       "0123456789abcdef0123456789abcdef",
		"ACCESS_TOKEN_TTL":        "15m",
		"REFRESH_TOKEN_TTL":       "168h",
		"CORS_ALLOWED_ORIGINS":    "https://fleet.example.com",
		"TRUSTED_PROXIES":         "127.0.0.1/32,10.0.0.1",
	}
	cfg, err := Parse(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	serialized := make([]string, 0, len(cfg.Redacted()))
	for key, value := range cfg.Redacted() {
		serialized = append(serialized, key+"="+toString(value))
	}
	all := strings.Join(serialized, " ")
	for _, secret := range []string{"database-secret", "redis-secret", "rabbit-secret", values["JWT_ACCESS_SECRET"]} {
		if strings.Contains(all, secret) {
			t.Fatalf("Redacted() leaked %q", secret)
		}
	}
}

func TestParseRejectsUnsafeConfiguration(t *testing.T) {
	values := map[string]string{
		"APP_ENV": "production",
	}
	_, err := Parse(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err == nil || !strings.Contains(err.Error(), "JWT_ACCESS_SECRET") {
		t.Fatalf("Parse() error = %v, want aggregated validation error", err)
	}
}

func toString(value any) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(
		strings.TrimSpace(strings.Repeat(" ", 0))+formatValue(value), "\n", " "), "\t", " "))
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []string:
		return strings.Join(typed, ",")
	default:
		return ""
	}
}
