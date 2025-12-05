package eveauth

import (
	"net/http"
)

// LeveledLogger is an interface that can be implemented by any logger
// or a logger wrapper to provide leveled logging (e.g. slog)
// The methods accept a message string and a variadic number of key-value pairs.
type LeveledLogger interface {
	Error(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Debug(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
}

// responseInfo is a struct for holding response details.
type responseInfo struct {
	status int
}

func (i responseInfo) Status() int {
	if i.status == 0 {
		// Status will usually only be set if it differs from 200
		return http.StatusOK
	}
	return i.status
}

// loggingResponseWriter is a http.ResponseWriter with enables request logging.
type loggingResponseWriter struct {
	http.ResponseWriter // compose original http.ResponseWriter
	info                responseInfo
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.info.status = statusCode
}

// requestLogger is a middleware handler that logs incoming requests.
type requestLogger struct {
	handler http.Handler
	logger  LeveledLogger
}

// newRequestLogger returns a new RequestLogger.
func newRequestLogger(handlerToWrap http.Handler, logger LeveledLogger) *requestLogger {
	return &requestLogger{
		handler: handlerToWrap,
		logger:  logger,
	}
}

// ServeHTTP handles the request by passing it to the real
// handler and logging the request and response.
func (l *requestLogger) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	l.logger.Debug("sso-server request", "method", r.Method, "path", r.URL.Path)
	lw := &loggingResponseWriter{ResponseWriter: rw}
	l.handler.ServeHTTP(lw, r)
	l.logger.Info("sso-server response", "method", r.Method, "path", r.URL.Path, "status", lw.info.Status())
}
