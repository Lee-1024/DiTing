package enforcement

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	repository Repository
}

type deploymentRequest struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type emergencyDisableResponse struct {
	DisabledCount int    `json:"disabledCount"`
	Message       string `json:"message"`
}

func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request Policy
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validPolicy(request) {
		http.Error(w, "name, template and yaml are required", http.StatusBadRequest)
		return
	}
	created, err := h.repository.Create(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	policies, err := h.repository.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, policies)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	policy, err := h.repository.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var request Policy
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validPolicy(request) {
		http.Error(w, "name, template and yaml are required", http.StatusBadRequest)
		return
	}
	updated, err := h.repository.Update(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateDeployment(w http.ResponseWriter, r *http.Request) {
	var request deploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if !validDeploymentStatus(request.Status) {
		http.Error(w, "invalid deployment status", http.StatusBadRequest)
		return
	}
	updated, err := h.repository.UpdateDeployment(r.Context(), r.PathValue("id"), request.Status, request.Message)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) EmergencyDisable(w http.ResponseWriter, r *http.Request) {
	message := "紧急停用所有拦截策略"
	count, err := h.repository.EmergencyDisable(r.Context(), message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, emergencyDisableResponse{DisabledCount: count, Message: message})
}

func (h *Handler) UpsertDeployment(w http.ResponseWriter, r *http.Request) {
	var request Deployment
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.HostID == "" || !validDeploymentStatus(request.Status) {
		http.Error(w, "hostId and valid status are required", http.StatusBadRequest)
		return
	}
	request.PolicyID = r.PathValue("id")
	updated, err := h.repository.UpsertHostDeployment(r.Context(), request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	deployments, err := h.repository.ListHostDeployments(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deployments)
}

func validPolicy(policy Policy) bool {
	return policy.Name != "" && policy.Template != "" && policy.YAML != ""
}

func validDeploymentStatus(status string) bool {
	switch status {
	case "draft", "deployed", "failed", "disabled":
		return true
	default:
		return false
	}
}

func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "enforcement policy not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
