package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv               string
	ServiceName          string
	HTTPAddr             string
	LogLevel             string
	DatabaseURL          string
	RedisAddr            string
	RedisPassword        string
	RabbitMQURL          string
	JWTIssuer            string
	JWTAudience          string
	JWTAccessSecret      string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	CORSAllowedOrigins   []string
	TrustedProxies       []string
}

type LookupFunc func(string) (string, bool)

func Load() (Config, error) {
	return parse(os.LookupEnv, true)
}

func LoadBackground() (Config, error) {
	return parse(os.LookupEnv, false)
}

func Parse(lookup LookupFunc) (Config, error) {
	return parse(lookup, true)
}

func parse(lookup LookupFunc, requireHTTP bool) (Config, error) {
	var cfg Config
	var problems []string

	required := func(name string) string {
		value, ok := lookup(name)
		if !ok || strings.TrimSpace(value) == "" {
			problems = append(problems, name+" is required")
			return ""
		}
		return value
	}

	cfg.AppEnv = strings.ToLower(strings.TrimSpace(required("APP_ENV")))
	switch cfg.AppEnv {
	case "development", "test", "staging", "production":
	default:
		problems = append(problems, "APP_ENV must be development, test, staging, or production")
	}

	cfg.ServiceName = strings.TrimSpace(required("SERVICE_NAME"))
	if !validServiceName(cfg.ServiceName) {
		problems = append(problems, "SERVICE_NAME must contain 2-63 lowercase letters, digits, or hyphens")
	}

	if requireHTTP {
		cfg.HTTPAddr = strings.TrimSpace(required("HTTP_ADDR"))
		if cfg.HTTPAddr != "" {
			if err := validateHostPort(cfg.HTTPAddr, true); err != nil {
				problems = append(problems, "HTTP_ADDR "+err.Error())
			}
		}
	} else if value, ok := lookup("HTTP_ADDR"); ok && strings.TrimSpace(value) != "" {
		cfg.HTTPAddr = strings.TrimSpace(value)
		if err := validateHostPort(cfg.HTTPAddr, true); err != nil {
			problems = append(problems, "HTTP_ADDR "+err.Error())
		}
	}

	cfg.LogLevel = strings.ToLower(strings.TrimSpace(required("LOG_LEVEL")))
	switch cfg.LogLevel {
	case "trace", "debug", "info", "warn", "error":
	default:
		problems = append(problems, "LOG_LEVEL must be trace, debug, info, warn, or error")
	}

	cfg.DatabaseURL = required("DATABASE_URL")
	if cfg.DatabaseURL != "" {
		if err := validateURL(cfg.DatabaseURL, "postgres", "postgresql"); err != nil {
			problems = append(problems, "DATABASE_URL "+err.Error())
		}
	}

	cfg.RedisAddr = strings.TrimSpace(required("REDIS_ADDR"))
	if cfg.RedisAddr != "" {
		if err := validateHostPort(cfg.RedisAddr, false); err != nil {
			problems = append(problems, "REDIS_ADDR "+err.Error())
		}
	}
	cfg.RedisPassword = required("REDIS_PASSWORD")

	cfg.RabbitMQURL = required("RABBITMQ_URL")
	if cfg.RabbitMQURL != "" {
		if err := validateURL(cfg.RabbitMQURL, "amqp", "amqps"); err != nil {
			problems = append(problems, "RABBITMQ_URL "+err.Error())
		}
	}

	cfg.JWTIssuer = strings.TrimSpace(required("JWT_ISSUER"))
	cfg.JWTAudience = strings.TrimSpace(required("JWT_AUDIENCE"))
	cfg.JWTAccessSecret = required("JWT_ACCESS_SECRET")
	if len([]byte(cfg.JWTAccessSecret)) < 32 {
		problems = append(problems, "JWT_ACCESS_SECRET must contain at least 32 bytes")
	}
	if strings.TrimSpace(cfg.JWTAccessSecret) != cfg.JWTAccessSecret {
		problems = append(problems, "JWT_ACCESS_SECRET must not have leading or trailing whitespace")
	}

	cfg.AccessTokenTTL = parseDuration(required("ACCESS_TOKEN_TTL"), "ACCESS_TOKEN_TTL", &problems)
	cfg.RefreshTokenTTL = parseDuration(required("REFRESH_TOKEN_TTL"), "REFRESH_TOKEN_TTL", &problems)
	if cfg.AccessTokenTTL > 0 && cfg.RefreshTokenTTL > 0 && cfg.RefreshTokenTTL <= cfg.AccessTokenTTL {
		problems = append(problems, "REFRESH_TOKEN_TTL must be greater than ACCESS_TOKEN_TTL")
	}

	production := cfg.AppEnv == "production"
	cfg.CORSAllowedOrigins = parseOrigins(required("CORS_ALLOWED_ORIGINS"), production, &problems)
	cfg.TrustedProxies = parseProxies(required("TRUSTED_PROXIES"), production, &problems)

	if len(problems) > 0 {
		sort.Strings(problems)
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}
	return cfg, nil
}

func (c Config) Redacted() map[string]any {
	return map[string]any{
		"app_env":                 c.AppEnv,
		"service_name":            c.ServiceName,
		"http_addr":               c.HTTPAddr,
		"log_level":               c.LogLevel,
		"database_url":            "[REDACTED]",
		"redis_addr":              c.RedisAddr,
		"redis_password":          "[REDACTED]",
		"rabbitmq_url":            "[REDACTED]",
		"jwt_issuer":              c.JWTIssuer,
		"jwt_audience":            c.JWTAudience,
		"jwt_access_secret":       "[REDACTED]",
		"access_token_ttl":        c.AccessTokenTTL.String(),
		"refresh_token_ttl":       c.RefreshTokenTTL.String(),
		"cors_allowed_origins":    append([]string(nil), c.CORSAllowedOrigins...),
		"trusted_proxies":         append([]string(nil), c.TrustedProxies...),
	}
}

func validServiceName(value string) bool {
	if len(value) < 2 || len(value) > 63 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			continue
		}
		return false
	}
	return true
}

func validateHostPort(value string, allowEmptyHost bool) error {
	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("must be a host:port pair")
	}
	if !allowEmptyHost && strings.TrimSpace(host) == "" {
		return fmt.Errorf("must include a host")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("must include a port from 1 through 65535")
	}
	return nil
}

func validateURL(value string, schemes ...string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("must be an absolute URL")
	}
	for _, scheme := range schemes {
		if parsed.Scheme == scheme {
			return nil
		}
	}
	return fmt.Errorf("has unsupported scheme %q", parsed.Scheme)
}

func parseDuration(value, name string, problems *[]string) time.Duration {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration <= 0 {
		*problems = append(*problems, name+" must be a positive Go duration")
		return 0
	}
	return duration
}

func parseOrigins(value string, production bool, problems *[]string) []string {
	parts := splitCSV(value)
	if len(parts) == 0 {
		*problems = append(*problems, "CORS_ALLOWED_ORIGINS must contain at least one origin")
		return nil
	}
	seen := make(map[string]struct{}, len(parts))
	result := make([]string, 0, len(parts))
	for _, origin := range parts {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			*problems = append(*problems, "CORS_ALLOWED_ORIGINS contains invalid origin "+origin)
			continue
		}
		if production && parsed.Scheme != "https" {
			*problems = append(*problems, "CORS_ALLOWED_ORIGINS must use https in production: "+origin)
			continue
		}
		if production && isLocalOriginHost(parsed.Hostname()) {
			*problems = append(*problems, "CORS_ALLOWED_ORIGINS must not use localhost or a loopback address in production: "+origin)
			continue
		}
		if _, exists := seen[origin]; !exists {
			seen[origin] = struct{}{}
			result = append(result, origin)
		}
	}
	return result
}

func parseProxies(value string, production bool, problems *[]string) []string {
	parts := splitCSV(value)
	if len(parts) == 0 {
		*problems = append(*problems, "TRUSTED_PROXIES must contain at least one IP address or CIDR")
		return nil
	}
	result := make([]string, 0, len(parts))
	for _, proxy := range parts {
		if _, network, err := net.ParseCIDR(proxy); err == nil {
			if production && !isNarrowTrustedProxy(network) {
				*problems = append(*problems, "TRUSTED_PROXIES must use IPv4 /24 or narrower or IPv6 /64 or narrower in production: "+proxy)
				continue
			}
			result = append(result, network.String())
			continue
		}
		ip := net.ParseIP(proxy)
		if ip == nil {
			*problems = append(*problems, "TRUSTED_PROXIES contains invalid address "+proxy)
			continue
		}
		if ip.To4() != nil {
			result = append(result, ip.String()+"/32")
		} else {
			result = append(result, ip.String()+"/128")
		}
	}
	return result
}

func isLocalOriginHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

func isNarrowTrustedProxy(network *net.IPNet) bool {
	if network == nil || network.IP.IsUnspecified() {
		return false
	}
	ones, bits := network.Mask.Size()
	return (bits == 32 && ones >= 24) || (bits == 128 && ones >= 64)
}

func splitCSV(value string) []string {
	var result []string
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
