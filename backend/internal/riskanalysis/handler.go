package riskanalysis

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

type Handler struct {
	repository      Repository
	auditRepository audit.Repository
	analyzer        Analyzer
}

func NewHandler(repository Repository, auditRepository audit.Repository, analyzer Analyzer) *Handler {
	return &Handler{repository: repository, auditRepository: auditRepository, analyzer: analyzer}
}

func (h *Handler) BatchGet(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EventIDs []string `json:"eventIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := h.repository.GetByEventIDs(r.Context(), request.EventIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(r.PathValue("event_id"))
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	if h.analyzer == nil {
		http.Error(w, "AI 分析服务未配置", http.StatusServiceUnavailable)
		return
	}
	event, err := h.auditRepository.GetEvent(r.Context(), eventID)
	if err != nil {
		if err == audit.ErrNotFound {
			http.Error(w, "审计事件不存在", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	started := time.Now()
	analysis, err := h.analyzer.Analyze(r.Context(), event)
	if err != nil {
		slog.Error("ai risk analysis failed", "event_id", eventID, "duration_ms", time.Since(started).Milliseconds(), "error", err)
		http.Error(w, err.Error(), aiAnalysisErrorStatus(err))
		return
	}
	analysis, err = h.repository.Upsert(r.Context(), analysis)
	if err != nil {
		slog.Error("save ai risk analysis failed", "event_id", eventID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(analysis)
}

func aiAnalysisErrorStatus(err error) int {
	if errorsIsTimeout(err) {
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

func errorsIsTimeout(err error) bool {
	return err != nil && (os.IsTimeout(err) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "Client.Timeout"))
}
