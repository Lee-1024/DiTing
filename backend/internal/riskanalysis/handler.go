package riskanalysis

import (
	"encoding/json"
	"net/http"
	"strings"

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
		http.Error(w, "ai analyzer is not configured", http.StatusServiceUnavailable)
		return
	}
	event, err := h.auditRepository.GetEvent(r.Context(), eventID)
	if err != nil {
		if err == audit.ErrNotFound {
			http.Error(w, "audit event not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	analysis, err := h.analyzer.Analyze(r.Context(), event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	analysis, err = h.repository.Upsert(r.Context(), analysis)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(analysis)
}
