package notification

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"diting/backend/internal/auth"
)

type Handler struct {
	repository Repository
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := h.repository.List(r.Context(), claims.UserID, r.URL.Query().Get("view"), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "notification id is required", http.StatusBadRequest)
		return
	}
	if err := h.repository.MarkRead(r.Context(), claims.UserID, id); err != nil {
		writeRepositoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.repository.MarkAllRead(r.Context(), claims.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "notification id is required", http.StatusBadRequest)
		return
	}
	var request struct {
		Disposition string `json:"disposition"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if _, err := NormalizeDisposition(request.Disposition); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.repository.Handle(r.Context(), id, request.Disposition, claims.Username); err != nil {
		writeRepositoryError(w, err)
		return
	}
	if err := h.repository.MarkRead(r.Context(), claims.UserID, id); err != nil {
		writeRepositoryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeRepositoryError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, ErrInvalidDisposition) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
