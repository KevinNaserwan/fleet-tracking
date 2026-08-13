package lifecycle

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KevinNaserwan/fleet-tracking/internal/config"
	"github.com/KevinNaserwan/fleet-tracking/internal/logging"
)

func Run(component string) int {
	cfg, err := config.LoadBackground()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logger, err := logging.New(cfg.LogLevel, cfg.AppEnv, cfg.ServiceName)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	logger.Warn().
		Str("component", component).
		Str("milestone", "1").
		Str("status", "pending").
		Msg("process lifecycle is active; feature processing is not implemented in milestone 1")

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	<-ctx.Done()
	logger.Info().Str("component", component).Msg("shutdown signal received")
	return 0
}
