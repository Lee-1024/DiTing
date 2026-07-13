package rule

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	repository Repository
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request Rule
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if !validSeverity(request.Severity) {
		http.Error(w, "invalid severity", http.StatusBadRequest)
		return
	}

	created, err := h.repository.Create(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rules, err := h.repository.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rules)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rule, err := h.repository.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeRuleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rule)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var request Rule
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if !validSeverity(request.Severity) {
		http.Error(w, "invalid severity", http.StatusBadRequest)
		return
	}

	updated, err := h.repository.Update(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeRuleError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeRuleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeRuleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func validSeverity(value string) bool {
	switch value {
	case "info", "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}
