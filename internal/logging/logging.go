package logging

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

func New(levelText, environment, service string) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(levelText)
	if err != nil {
		return zerolog.Logger{}, err
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	logger := zerolog.New(zerolog.SyncWriter(os.Stdout))
	logger = logger.Level(level).With().
		Timestamp().
		Str("service", service).
		Str("environment", environment).
		Logger()
	return logger, nil
}

type StdlibWriter struct {
	Logger zerolog.Logger
}

func (w StdlibWriter) Write(p []byte) (int, error) {
	message := strings.TrimSpace(string(p))
	if message != "" {
		w.Logger.Error().Str("source", "net/http").Msg(message)
	}
	return len(p), nil
}
