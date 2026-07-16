package useradmin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct {
	repository Repository
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repository.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var request CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := validateCreateUser(request); err != nil {
		writeUserAdminError(w, err)
		return
	}
	created, err := h.repository.CreateUser(r.Context(), request)
	if err != nil {
		writeUserAdminError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	var request UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := validateUpdateUser(request); err != nil {
		writeUserAdminError(w, err)
		return
	}
	updated, err := h.repository.UpdateUser(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeUserAdminError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(request.Password) < 6 {
		writeUserAdminError(w, ErrWeakPassword)
		return
	}
	if err := h.repository.ResetPassword(r.Context(), r.PathValue("id"), request.Password); err != nil {
		writeUserAdminError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.DeleteUser(r.Context(), r.PathValue("id")); err != nil {
		writeUserAdminError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.repository.ListRoles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(roles)
}

func validateCreateUser(request CreateUserRequest) error {
	if strings.TrimSpace(request.Username) == "" || strings.TrimSpace(request.DisplayName) == "" {
		return ErrInvalidRequest
	}
	if len(request.Password) < 6 {
		return ErrWeakPassword
	}
	if !validStatus(request.Status) {
		return ErrInvalidRequest
	}
	return nil
}

func validateUpdateUser(request UpdateUserRequest) error {
	if strings.TrimSpace(request.DisplayName) == "" || !validStatus(request.Status) {
		return ErrInvalidRequest
	}
	return nil
}

func validStatus(value string) bool {
	return value == "" || value == "active" || value == "disabled"
}

func writeUserAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		http.Error(w, "invalid user request", http.StatusBadRequest)
	case errors.Is(err, ErrWeakPassword):
		http.Error(w, ErrWeakPassword.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrConflict):
		http.Error(w, "user already exists", http.StatusConflict)
	case errors.Is(err, ErrLastAdmin):
		http.Error(w, ErrLastAdmin.Error(), http.StatusConflict)
	case errors.Is(err, ErrRoleNotFound):
		http.Error(w, ErrRoleNotFound.Error(), http.StatusBadRequest)
	case errors.Is(err, ErrNotFound):
		http.Error(w, "user not found", http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
