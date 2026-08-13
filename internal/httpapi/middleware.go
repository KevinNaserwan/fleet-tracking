package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KevinNaserwan/fleet-tracking/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

const requestIDKey = "request_id"

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:/-]{1,128}$`)

func RequestID(c *gin.Context) string {
	value, _ := c.Get(requestIDKey)
	requestID, _ := value.(string)
	return requestID
}

func RequestIDMiddleware(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !requestIDPattern.MatchString(requestID) {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		requestLogger := logger.With().Str("request_id", requestID).Logger()
		c.Request = c.Request.WithContext(requestLogger.WithContext(c.Request.Context()))
		c.Next()
	}
}

func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		status := c.Writer.Status()
		logger := zerolog.Ctx(c.Request.Context())
		event := logger.Info()
		if status >= http.StatusInternalServerError {
			event = logger.Error()
		} else if status >= http.StatusBadRequest {
			event = logger.Warn()
		}
		event.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Int("status", status).
			Int("response_bytes", c.Writer.Size()).
			Dur("duration", time.Since(started)).
			Msg("http request")
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				zerolog.Ctx(c.Request.Context()).Error().
					Interface("panic", recovered).
					Bytes("stack", debug.Stack()).
					Msg("panic recovered")
				if !c.Writer.Written() {
					WriteError(
						c,
						http.StatusInternalServerError,
						"internal_error",
						"An unexpected error occurred.",
					)
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Header("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		c.Next()
	}
}

func CORSMiddleware(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		c.Header("Vary", "Origin")
		if _, ok := allowed[origin]; !ok {
			if c.Request.Method == http.MethodOptions {
				WriteError(c, http.StatusForbidden, "forbidden", "Origin is not allowed.")
				c.Abort()
				return
			}
			c.Next()
			return
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "600")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	}
}

func BodyLimitMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			if c.Request.ContentLength > limit {
				WriteError(
					c,
					http.StatusBadRequest,
					"bad_request",
					"Request body exceeds the allowed size.",
				)
				c.Abort()
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewMetrics(registerer prometheus.Registerer) *Metrics {
	metrics := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "fleet_tracking",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests handled.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "fleet_tracking",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}
	registerer.MustRegister(metrics.requests, metrics.duration)
	return metrics
}

func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		m.requests.WithLabelValues(c.Request.Method, route, status).Inc()
		m.duration.WithLabelValues(c.Request.Method, route, status).
			Observe(time.Since(started).Seconds())
	}
}

type rateEntry struct {
	windowStarted time.Time
	count         int
}

type LoginRateLimiter struct {
	mu         sync.Mutex
	entries    map[string]rateEntry
	limit      int
	window     time.Duration
	maxEntries int
	message    string
	operations uint64
	now        func() time.Time
}

const (
	rateLimiterMaxEntries      = 10_000
	rateLimiterCleanupInterval = 64
	rateLimiterCleanupBudget   = 64
)

func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return newRateLimiter(limit, window, "Too many login attempts. Try again later.")
}

func NewRefreshRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return newRateLimiter(limit, window, "Too many refresh attempts. Try again later.")
}

func newRateLimiter(limit int, window time.Duration, message string) *LoginRateLimiter {
	return &LoginRateLimiter{
		entries: make(map[string]rateEntry),
		limit: limit, window: window, maxEntries: rateLimiterMaxEntries,
		message: message, now: time.Now,
	}
}

func (l *LoginRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, retryAfter := l.allow(c.ClientIP())
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			WriteError(
					c,
					http.StatusTooManyRequests,
					"rate_limited",
					l.message,
				)
			c.Abort()
			return
		}
		c.Next()
	}
}

func (l *LoginRateLimiter) allow(key string) (bool, int) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.operations++
	if l.operations%rateLimiterCleanupInterval == 0 {
		l.cleanup(now, rateLimiterCleanupBudget)
	}
	entry, exists := l.entries[key]
	if !exists || now.Sub(entry.windowStarted) >= l.window {
		if !exists && len(l.entries) >= l.maxEntries {
			l.cleanup(now, rateLimiterCleanupBudget)
			if len(l.entries) >= l.maxEntries {
				return false, max(1, int(math.Ceil(l.window.Seconds())))
			}
		}
		l.entries[key] = rateEntry{windowStarted: now, count: 1}
		return true, 0
	}
	if entry.count >= l.limit {
		seconds := int(math.Ceil(l.window.Seconds() - now.Sub(entry.windowStarted).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		return false, seconds
	}
	entry.count++
	l.entries[key] = entry
	return true, 0
}

func (l *LoginRateLimiter) cleanup(now time.Time, budget int) {
	examined := 0
	for key, entry := range l.entries {
		if examined >= budget {
			return
		}
		examined++
		if now.Sub(entry.windowStarted) >= l.window {
			delete(l.entries, key)
		}
	}
}

func Authenticate(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c)
			return
		}
		principal, err := tokens.Parse(parts[1])
		if err != nil {
			unauthorized(c)
			return
		}
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFromContext(c.Request.Context())
		if !ok {
			unauthorized(c)
			return
		}
		for _, granted := range principal.Permissions {
			if granted == permission {
				c.Next()
				return
			}
		}
		WriteError(
			c,
			http.StatusForbidden,
			"forbidden",
			fmt.Sprintf("Permission %q is required.", permission),
		)
		c.Abort()
	}
}

func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer realm="fleet-tracking"`)
	WriteError(c, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
	c.Abort()
}

func newRequestID() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "req_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return "req_" + hex.EncodeToString(random)
}
