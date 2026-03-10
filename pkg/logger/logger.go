package logger

import (
	"context"

	"github.com/ialexeze/multi-crd-controller/pkg/config/pkg/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Init(cfg *config.Config) {
	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch {
	case cfg.IsDev():
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case cfg.IsStaging():
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case cfg.IsProduction():
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func FromContext(ctx context.Context) *zerolog.Logger {
	l := log.With()

	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		l = l.Str(RequestIDKey.String(), reqID)
	}

	if crd, ok := ctx.Value(CRDKey).(string); ok {
		l = l.Str(CRDKey.String(), crd)
	}

	if res, ok := ctx.Value(ResourceKey).(string); ok {
		l = l.Str(ResourceKey.String(), res)
	}

	logger := l.Logger()
	return &logger
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Info() *zerolog.Event {
	return log.Info()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}

func Panic() *zerolog.Event {
	return log.Panic()
}
