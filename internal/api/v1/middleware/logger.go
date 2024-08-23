package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

type Logger struct {
	logger *slog.Logger
}

type StatusResponseWriter struct {
	responseWriter http.ResponseWriter
	statusCode     int
}

func (w *StatusResponseWriter) Header() http.Header {
	return w.responseWriter.Header()
}

func (w *StatusResponseWriter) Write(b []byte) (int, error) {
	return w.responseWriter.Write(b)
}

func (w *StatusResponseWriter) WriteHeader(statusCode int) {
	w.responseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func (l *Logger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := l.logger.With("method", r.Method, "path", r.URL.Path)

		statusResponseWriter := StatusResponseWriter{
			responseWriter: w,
			statusCode:     0,
		}

		ctx := ContextWithLogger(r.Context(), logger)

		r = r.Clone(ctx)

		next.ServeHTTP(&statusResponseWriter, r)

		logger.Debug("debug", "code", statusResponseWriter.statusCode)
	})
}

func ContextWithLogger(parent context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(parent, log, logger)
}

func LoggerFrom(ctx context.Context) *slog.Logger {
	v := ctx.Value(log)
	if v == nil {
		return nil
	}

	logger, ok := v.(*slog.Logger)
	if !ok {
		return nil
	}

	return logger
}

func NewLogger(logger *slog.Logger) *Logger {
	l := Logger{
		logger: logger,
	}

	return &l
}
