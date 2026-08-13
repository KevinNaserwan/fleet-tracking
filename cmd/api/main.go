package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/KevinNaserwan/fleet-tracking/internal/auth"
	"github.com/KevinNaserwan/fleet-tracking/internal/config"
	"github.com/KevinNaserwan/fleet-tracking/internal/httpapi"
	"github.com/KevinNaserwan/fleet-tracking/internal/logging"
	"github.com/KevinNaserwan/fleet-tracking/internal/platform"
	"github.com/KevinNaserwan/fleet-tracking/internal/store"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "healthcheck" {
		if err := healthcheck(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: api [healthcheck]")
		os.Exit(2)
	}
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := logging.New(cfg.LogLevel, cfg.AppEnv, cfg.ServiceName)
	if err != nil {
		return err
	}
	logger.Info().Interface("configuration", cfg.Redacted()).Msg("configuration loaded")

	rootContext, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	dependencies, err := platform.Open(rootContext, cfg)
	if err != nil {
		return fmt.Errorf("initialize dependencies: %w", err)
	}
	defer dependencies.Close()

	tokens, err := auth.NewTokenManager(
		cfg.JWTIssuer,
		cfg.JWTAudience,
		cfg.JWTAccessSecret,
		cfg.AccessTokenTTL,
	)
	if err != nil {
		return err
	}
	repository := store.NewPostgres(dependencies.DB)
	authService := auth.NewService(repository, tokens, cfg.RefreshTokenTTL)
	router, err := httpapi.NewRouter(cfg, logger, dependencies, authService, tokens)
	if err != nil {
		return fmt.Errorf("initialize router: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          stdlog.New(logging.StdlibWriter{Logger: logger}, "", 0),
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info().Str("address", cfg.HTTPAddr).Msg("API server listening")
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-rootContext.Done():
		logger.Info().Msg("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}
	logger.Info().Msg("API server stopped")
	return nil
}

func healthcheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	httpAddr := strings.TrimSpace(os.Getenv("HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = ":8080"
	}
	_, port, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return fmt.Errorf("invalid HTTP_ADDR for healthcheck: %w", err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("invalid HTTP_ADDR port for healthcheck")
	}
	probeURL := "http://" + net.JoinHostPort("127.0.0.1", port) + "/health/live"
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		probeURL,
		nil,
	)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("liveness request: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("liveness returned HTTP %d", response.StatusCode)
	}
	return nil
}
