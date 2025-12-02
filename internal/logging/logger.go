package logging

import (
	"go.uber.org/zap"
)

// Logger is our app-wide logging abstraction.
// We use key-value style args similar to zap.SugaredLogger.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	With(args ...any) Logger
}

type zapLogger struct {
	s *zap.SugaredLogger
}

// New creates a JSON logger with service + env fields pre-attached.
func New(serviceName, env string) Logger {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.OutputPaths = []string{"stdout"}

	core, err := cfg.Build()
	if err != nil {
		// In a starter template, it's fine to panic here; in prod you'd handle this more gracefully.
		panic(err)
	}

	s := core.Sugar().With(
		"service", serviceName,
		"env", env,
	)

	return &zapLogger{s: s}
}

func (l *zapLogger) Info(msg string, args ...any) {
	l.s.Infow(msg, args...)
}

func (l *zapLogger) Error(msg string, args ...any) {
	l.s.Errorw(msg, args...)
}

func (l *zapLogger) Debug(msg string, args ...any) {
	l.s.Debugw(msg, args...)
}

func (l *zapLogger) With(args ...any) Logger {
	return &zapLogger{s: l.s.With(args...)}
}

// AsZap unwraps our Logger to a *zap.Logger for integrations (Watermill, OTel, etc.).
// If someone passes a different Logger implementation, we fall back to a no-op logger.
func AsZap(l Logger) *zap.Logger {
	if zl, ok := l.(*zapLogger); ok {
		return zl.s.Desugar()
	}
	return zap.NewNop()
}
