package server

import (
	"log/slog"
	"net/http"
	"time"

	"diting/backend/internal/audit"
	"diting/backend/internal/auth"
	"diting/backend/internal/hostasset"
	"diting/backend/internal/operationlog"
	"diting/backend/internal/riskstatus"
	"diting/backend/internal/rule"
	"diting/backend/internal/stats"
)

func NewRouter(repository audit.Repository, ruleRepository rule.Repository, statsRepository stats.Repository, authService *auth.Service, operationRepository operationlog.Repository, hostAssetRepository hostasset.Repository, riskStatusRepository riskstatus.Repository) http.Handler {
	mux := http.NewServeMux()
	auditHandler := audit.NewHandler(repository)
	if ruleRepository == nil {
		ruleRepository = rule.NewMemoryRepository()
	}
	ruleHandler := rule.NewHandler(ruleRepository)
	if hostAssetRepository == nil {
		hostAssetRepository = hostasset.NewMemoryRepository()
	}
	hostAssetHandler := hostasset.NewHandler(hostAssetRepository)
	var riskStatusHandler *riskstatus.Handler
	if riskStatusRepository != nil {
		riskStatusHandler = riskstatus.NewHandler(riskStatusRepository)
	}
	statsHandler := stats.NewHandler(statsRepository)
	var authHandler *auth.Handler
	var protect func(http.Handler) http.Handler
	if authService != nil {
		authHandler = auth.NewHandler(authService)
		authMiddleware := auth.Middleware(authService)
		operationMiddleware := operationlog.Middleware(operationRepository)
		protect = func(next http.Handler) http.Handler {
			return authMiddleware(operationMiddleware(next))
		}
	} else {
		protect = func(next http.Handler) http.Handler { return next }
	}

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	if authHandler != nil {
		mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
		mux.Handle("/api/v1/auth/me", protect(http.HandlerFunc(authHandler.Me)))
		mux.Handle("/api/v1/auth/password", protect(http.HandlerFunc(authHandler.ChangePassword)))
	}
	mux.Handle("/api/v1/audit/events", protect(http.HandlerFunc(auditHandler.ListEvents)))
	mux.Handle("/api/v1/audit/events/export", protect(http.HandlerFunc(auditHandler.ExportEvents)))
	if riskStatusHandler != nil {
		mux.Handle("/api/v1/risk-dispositions/batch", protect(http.HandlerFunc(riskStatusHandler.BatchGet)))
		mux.Handle("/api/v1/risk-dispositions/{event_id}", protect(http.HandlerFunc(riskStatusHandler.Upsert)))
	}
	mux.Handle("/api/v1/rules", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ruleHandler.List(w, r)
		case http.MethodPost:
			ruleHandler.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/v1/rules/{id}", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ruleHandler.Get(w, r)
		case http.MethodPut:
			ruleHandler.Update(w, r)
		case http.MethodDelete:
			ruleHandler.Delete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/v1/host-assets", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			hostAssetHandler.List(w, r)
		case http.MethodPost:
			hostAssetHandler.Create(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/api/v1/host-assets/{id}", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			hostAssetHandler.Get(w, r)
		case http.MethodPut:
			hostAssetHandler.Update(w, r)
		case http.MethodDelete:
			hostAssetHandler.Delete(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	if statsRepository != nil {
		mux.Handle("/api/v1/stats/overview", protect(http.HandlerFunc(statsHandler.Overview)))
		mux.Handle("/api/v1/stats/event-trend", protect(http.HandlerFunc(statsHandler.EventTrend)))
		mux.Handle("/api/v1/stats/top-commands", protect(http.HandlerFunc(statsHandler.TopCommands)))
		mux.Handle("/api/v1/stats/commands", protect(http.HandlerFunc(statsHandler.CommandStats)))
		mux.Handle("/api/v1/stats/commands/export", protect(http.HandlerFunc(statsHandler.ExportCommandStats)))
		mux.Handle("/api/v1/stats/users", protect(http.HandlerFunc(statsHandler.UserAudits)))
		mux.Handle("/api/v1/stats/hosts", protect(http.HandlerFunc(statsHandler.HostAudits)))
	}
	return loggingMiddleware(mux)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
