package sso

import (
	"log/slog"
	"net/http"
)

// struct for holding response details
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

// our http.ResponseWriter implementation
type loggingResponseWriter struct {
	http.ResponseWriter // compose original http.ResponseWriter
	info                responseInfo
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.info.status = statusCode
}

// Logger is a middleware handler that does request logging
type Logger struct {
	handler http.Handler
}

// WithLogger constructs a new Logger middleware handler
func WithLogger(handlerToWrap http.Handler) *Logger {
	return &Logger{handlerToWrap}
}

// ServeHTTP handles the request by passing it to the real
// handler and logging the request and response.
func (l *Logger) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	slog.Debug("SSO server request", "method", r.Method, "path", r.URL.Path)
	lw := &loggingResponseWriter{ResponseWriter: rw}
	l.handler.ServeHTTP(lw, r)
	slog.Info("SSO server response", "method", r.Method, "path", r.URL.Path, "status", lw.info.Status())
}
