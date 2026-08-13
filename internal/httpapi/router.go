package httpapi

import (
	"net/http"
	"time"

	"github.com/KevinNaserwan/fleet-tracking/internal/auth"
	"github.com/KevinNaserwan/fleet-tracking/internal/config"
	"github.com/KevinNaserwan/fleet-tracking/internal/platform"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

func NewRouter(
	cfg config.Config,
	logger zerolog.Logger,
	dependencies *platform.Dependencies,
	authService *auth.Service,
	tokens *auth.TokenManager,
) (*gin.Engine, error) {
	if cfg.AppEnv == "development" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.RedirectTrailingSlash = false
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector(), prometheus.NewProcessCollector(
		prometheus.ProcessCollectorOpts{},
	))
	metrics := NewMetrics(registry)

	router.Use(
		RequestIDMiddleware(logger),
		AccessLogMiddleware(),
		metrics.Middleware(),
		RecoveryMiddleware(),
		SecurityHeadersMiddleware(),
		CORSMiddleware(cfg.CORSAllowedOrigins),
		BodyLimitMiddleware(1<<20),
	)

	handler := NewHandler(authService, dependencies, logger, cfg.ServiceName)
	loginLimiter := NewLoginRateLimiter(10, time.Minute)
	refreshLimiter := NewRefreshRateLimiter(30, time.Minute)

	router.GET("/health/live", handler.Liveness)
	router.GET("/health/ready", handler.Readiness)
	// The reverse proxy keeps this unauthenticated scrape endpoint private.
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	)))

	v1 := router.Group("/api/v1")
	v1.POST("/auth/login", loginLimiter.Middleware(), handler.Login)
	v1.POST("/auth/refresh", refreshLimiter.Middleware(), handler.Refresh)
	v1.POST("/auth/logout", Authenticate(tokens), handler.Logout)
	v1.GET("/me", Authenticate(tokens), handler.Me)

	router.NoRoute(func(c *gin.Context) {
		WriteError(c, http.StatusNotFound, "not_found", "The requested resource was not found.")
	})
	router.NoMethod(func(c *gin.Context) {
		WriteError(c, http.StatusMethodNotAllowed, "method_not_allowed", "Method is not allowed.")
	})
	return router, nil
}
