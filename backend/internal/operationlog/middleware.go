package operationlog

import (
	"log/slog"
	"net"
	"net/http"
	"strings"

	"diting/backend/internal/auth"
)

func Middleware(repository Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			if repository == nil {
				return
			}
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				return
			}
			if err := repository.Create(r.Context(), Entry{
				UserID:    claims.UserID,
				Username:  claims.Username,
				Method:    r.Method,
				Path:      r.URL.Path,
				Status:    recorder.status,
				IP:        clientIP(r),
				UserAgent: r.UserAgent(),
			}); err != nil {
				slog.Error("operation log write failed", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "username", claims.Username, "error", err)
			}
		})
	}
}

func clientIP(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		if first, _, ok := strings.Cut(forwardedFor, ","); ok {
			return strings.TrimSpace(first)
		}
		return forwardedFor
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
