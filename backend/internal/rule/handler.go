package rule

import (
	"encoding/json"
	"errors"
	"net/http"

	"diting/backend/internal/audit"
)

type Handler struct {
	repository Repository
}

type TestRequest struct {
	Rule  Rule      `json:"rule"`
	Event TestEvent `json:"event"`
}

type TestEvent struct {
	EventType     string `json:"eventType"`
	Action        string `json:"action"`
	Severity      string `json:"severity"`
	HostID        string `json:"hostId"`
	HostName      string `json:"hostName"`
	NodeName      string `json:"nodeName"`
	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerID   string `json:"containerId"`
	ProcessName   string `json:"processName"`
	BinaryPath    string `json:"binaryPath"`
	Cmdline       string `json:"cmdline"`
	Username      string `json:"username"`
	LoginUsername string `json:"loginUsername"`
	FilePath      string `json:"filePath"`
	FileOperation string `json:"fileOperation"`
	DstIP         string `json:"dstIp"`
	DstPort       uint16 `json:"dstPort"`
	Protocol      string `json:"protocol"`
	Domain        string `json:"domain"`
}

type TestResponse struct {
	Matched bool              `json:"matched"`
	Message string            `json:"message"`
	Matches []audit.RuleMatch `json:"matches"`
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

func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	var request TestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.Rule.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if !validSeverity(request.Rule.Severity) {
		http.Error(w, "invalid severity", http.StatusBadRequest)
		return
	}

	matches := MatchConditions(request.Rule.MatchExpr, request.Event.toAuditEvent())
	response := TestResponse{
		Matched: len(matches) > 0,
		Message: "未命中",
		Matches: matches,
	}
	if response.Matched {
		response.Message = "命中"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (e TestEvent) toAuditEvent() audit.Event {
	return audit.Event{
		EventType:     e.EventType,
		Action:        e.Action,
		Severity:      e.Severity,
		HostID:        e.HostID,
		HostName:      e.HostName,
		NodeName:      e.NodeName,
		Namespace:     e.Namespace,
		PodName:       e.PodName,
		ContainerID:   e.ContainerID,
		ProcessName:   e.ProcessName,
		BinaryPath:    e.BinaryPath,
		Cmdline:       e.Cmdline,
		Username:      e.Username,
		LoginUsername: e.LoginUsername,
		FilePath:      e.FilePath,
		FileOperation: e.FileOperation,
		DstIP:         e.DstIP,
		DstPort:       e.DstPort,
		Protocol:      e.Protocol,
		Domain:        e.Domain,
	}
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
