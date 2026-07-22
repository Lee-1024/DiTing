package riskstatus

import (
	"encoding/json"
	"errors"
	"net/http"

	"diting/backend/internal/auth"
)

type Handler struct {
	repository Repository
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) BatchGet(w http.ResponseWriter, r *http.Request) {
	var request struct {
		EventIDs []string `json:"eventIds"`
		Events   []struct {
			EventID     string `json:"eventId"`
			Fingerprint string `json:"fingerprint"`
		} `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := h.repository.ListByEventIDs(r.Context(), request.EventIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fingerprintsByEventID := map[string]string{}
	fingerprints := []string{}
	for _, event := range request.Events {
		if event.EventID == "" || event.Fingerprint == "" {
			continue
		}
		fingerprintsByEventID[event.EventID] = event.Fingerprint
		fingerprints = append(fingerprints, event.Fingerprint)
	}
	ignored, err := h.repository.ListByFingerprints(r.Context(), fingerprints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for eventID, fingerprint := range fingerprintsByEventID {
		if _, exists := result[eventID]; exists {
			continue
		}
		if disposition, ignored := ignored[fingerprint]; ignored {
			disposition.EventID = eventID
			result[eventID] = disposition
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Status      string `json:"status"`
		Note        string `json:"note"`
		Scope       string `json:"scope"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	claims, _ := auth.ClaimsFromContext(r.Context())
	disposition, err := h.repository.Upsert(r.Context(), Disposition{
		EventID:     r.PathValue("event_id"),
		Status:      request.Status,
		Note:        request.Note,
		Scope:       request.Scope,
		Fingerprint: request.Fingerprint,
		HandledBy:   claims.Username,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(disposition)
}
