package logger

import (
	"context"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	// Default to info so package-level init() calls (e.g. typeregistry)
	// don't emit debug logs before the CLI has a chance to parse --debug.
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Init(level string) {
	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
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
