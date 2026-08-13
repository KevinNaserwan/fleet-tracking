package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/KevinNaserwan/fleet-tracking/internal/auth"
	"github.com/KevinNaserwan/fleet-tracking/internal/platform"
	"github.com/KevinNaserwan/fleet-tracking/internal/version"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var tenantSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Handler struct {
	auth         *auth.Service
	dependencies *platform.Dependencies
	logger       zerolog.Logger
	serviceName  string
}

func NewHandler(
	authService *auth.Service,
	dependencies *platform.Dependencies,
	logger zerolog.Logger,
	serviceName string,
) *Handler {
	return &Handler{
		auth: authService, dependencies: dependencies, logger: logger, serviceName: serviceName,
	}
}

type loginRequest struct {
	TenantSlug string `json:"tenantSlug"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handler) Login(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request loginRequest
	if !decodeRequest(c, &request) {
		return
	}
	details := validateLogin(request)
	if len(details) > 0 {
		WriteError(
			c, http.StatusUnprocessableEntity, "validation_failed",
			"One or more fields are invalid.", details...,
		)
		return
	}
	result, err := h.auth.Login(c.Request.Context(), auth.LoginInput{
		TenantSlug: request.TenantSlug,
		Email:      strings.ToLower(request.Email),
		Password:   request.Password,
	}, clientMetadata(c))
	if err != nil {
		h.writeServiceError(c, err, "Invalid tenant, email, or password.")
		return
	}
	WriteSuccess(c, http.StatusOK, gin.H{"tokens": result.Tokens, "user": result.User})
}

func (h *Handler) Refresh(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request refreshRequest
	if !decodeRequest(c, &request) {
		return
	}
	if details := validateRefresh(request.RefreshToken); len(details) > 0 {
		WriteError(
			c, http.StatusUnprocessableEntity, "validation_failed",
			"One or more fields are invalid.", details...,
		)
		return
	}
	tokens, err := h.auth.Refresh(c.Request.Context(), request.RefreshToken, clientMetadata(c))
	if err != nil {
		h.writeServiceError(c, err, "Refresh token is invalid or expired.")
		return
	}
	WriteSuccess(c, http.StatusOK, tokens)
}

func (h *Handler) Logout(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request refreshRequest
	if !decodeRequest(c, &request) {
		return
	}
	if details := validateRefresh(request.RefreshToken); len(details) > 0 {
		WriteError(
			c, http.StatusUnprocessableEntity, "validation_failed",
			"One or more fields are invalid.", details...,
		)
		return
	}
	principal, ok := auth.PrincipalFromContext(c.Request.Context())
	if !ok {
		unauthorized(c)
		return
	}
	if err := h.auth.Logout(c.Request.Context(), principal, request.RefreshToken); err != nil {
		h.writeServiceError(c, err, "")
		return
	}
	WriteSuccess(c, http.StatusOK, gin.H{"loggedOut": true})
}

func (h *Handler) Me(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	principal, ok := auth.PrincipalFromContext(c.Request.Context())
	if !ok {
		unauthorized(c)
		return
	}
	user, err := h.auth.Me(c.Request.Context(), principal)
	if err != nil {
		h.writeServiceError(c, err, "Authentication is required.")
		return
	}
	WriteSuccess(c, http.StatusOK, user)
}

func (h *Handler) Liveness(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	WriteSuccess(c, http.StatusOK, gin.H{
		"status":    "ok",
		"service":   h.serviceName,
		"version":   version.Current,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

type dependencyCheck struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latencyMs"`
	Message   string `json:"message,omitempty"`
}

func (h *Handler) Readiness(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	checks := make(map[string]dependencyCheck, 3)
	var checksMu sync.Mutex
	var wait sync.WaitGroup
	run := func(name string, check func(context.Context) error) {
		defer wait.Done()
		started := time.Now()
		err := check(ctx)
		result := dependencyCheck{Status: "up", LatencyMS: time.Since(started).Milliseconds()}
		if err != nil {
			result.Status = "down"
			result.Message = "dependency unavailable"
			h.logger.Warn().Err(err).Str("dependency", name).Msg("readiness check failed")
		}
		checksMu.Lock()
		checks[name] = result
		checksMu.Unlock()
	}
	wait.Add(3)
	go run("database", func(ctx context.Context) error {
		return h.dependencies.DB.Ping(ctx)
	})
	go run("redis", func(ctx context.Context) error {
		return h.dependencies.Redis.Ping(ctx).Err()
	})
	go run("rabbitmq", h.dependencies.RabbitMQ.Check)
	wait.Wait()

	status := "ready"
	httpStatus := http.StatusOK
	for _, check := range checks {
		if check.Status == "down" {
			status = "not_ready"
			httpStatus = http.StatusServiceUnavailable
			break
		}
	}
	if httpStatus == http.StatusServiceUnavailable {
		c.Header("Retry-After", "5")
	}
	WriteSuccess(c, httpStatus, gin.H{
		"status": status, "service": h.serviceName, "version": version.Current,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "checks": checks,
	})
}

func (h *Handler) writeServiceError(c *gin.Context, err error, unauthorizedMessage string) {
	if errors.Is(err, auth.ErrUnauthorized) {
		WriteError(c, http.StatusUnauthorized, "unauthorized", unauthorizedMessage)
		return
	}
	h.logger.Error().Err(err).Str("request_id", RequestID(c)).Msg("request failed")
	WriteError(
		c,
		http.StatusInternalServerError,
		"internal_error",
		"An unexpected error occurred.",
	)
}

func decodeRequest(c *gin.Context, destination any) bool {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		WriteError(c, http.StatusBadRequest, "bad_request", "Content-Type must be application/json.")
		return false
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		WriteError(c, http.StatusBadRequest, "bad_request", "Request body is not valid JSON.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(c, http.StatusBadRequest, "bad_request", "Request body must contain one JSON object.")
		return false
	}
	return true
}

func validateLogin(request loginRequest) []ErrorDetail {
	var details []ErrorDetail
	if len(request.TenantSlug) < 1 || len(request.TenantSlug) > 63 ||
		!tenantSlugPattern.MatchString(request.TenantSlug) {
		details = append(details, ErrorDetail{
			Field: "tenantSlug", Code: "invalid_format",
			Message: "Must be a lowercase tenant slug.",
		})
	}
	if !validEmail(request.Email) {
		details = append(details, ErrorDetail{
			Field: "email", Code: "invalid_format", Message: "Must be a valid email address.",
		})
	}
	length := utf8.RuneCountInString(request.Password)
	if length < 8 || length > 1024 {
		details = append(details, ErrorDetail{
			Field: "password", Code: "invalid_length",
			Message: "Must contain between 8 and 1024 characters.",
		})
	}
	return details
}

func validateRefresh(token string) []ErrorDetail {
	length := utf8.RuneCountInString(token)
	if length < 32 || length > 2048 {
		return []ErrorDetail{{
			Field: "refreshToken", Code: "invalid_length",
			Message: "Must contain between 32 and 2048 characters.",
		}}
	}
	return nil
}

func validEmail(value string) bool {
	if value != strings.TrimSpace(value) || len(value) > 320 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && strings.Contains(value, "@")
}

func clientMetadata(c *gin.Context) auth.ClientMetadata {
	ip := c.ClientIP()
	if net.ParseIP(ip) == nil {
		ip = ""
	}
	return auth.ClientMetadata{IP: ip, UserAgent: truncateRunes(c.GetHeader("User-Agent"), 512)}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
