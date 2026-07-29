package riskstatus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"diting/backend/internal/auth"
	"diting/backend/internal/systemconfig"
)

type Handler struct {
	repository                Repository
	collectorFilterRepository systemconfig.Repository
}

// NewHandler 创建并初始化 New Handler 实例。
func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

// SetCollectorFilterRepository enables ignored-similar risk dispositions to suppress future collection.
func (h *Handler) SetCollectorFilterRepository(repository systemconfig.Repository) {
	h.collectorFilterRepository = repository
}

// List 查询并返回 List 列表。
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.repository.List(r.Context(), status, limit)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// BatchGet 处理 Batch Get 相关逻辑。
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

// Upsert 处理 Upsert 相关逻辑。
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Status      string `json:"status"`
		Note        string `json:"note"`
		Scope       string `json:"scope"`
		Fingerprint string `json:"fingerprint"`
		Event       struct {
			EventType     string   `json:"eventType"`
			Severity      string   `json:"severity"`
			Username      string   `json:"username"`
			LoginUsername string   `json:"loginUsername"`
			ProcessName   string   `json:"processName"`
			Cmdline       string   `json:"cmdline"`
			FilePath      string   `json:"filePath"`
			FileOperation string   `json:"fileOperation"`
			DstIP         string   `json:"dstIp"`
			DstPort       uint16   `json:"dstPort"`
			Protocol      string   `json:"protocol"`
			RuleIDs       []string `json:"ruleIds"`
		} `json:"event"`
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
	if disposition.Status == StatusIgnoreSimilar && h.collectorFilterRepository != nil {
		if err := h.addIgnoredSimilarCollectorRule(r.Context(), request.Fingerprint, request.Event); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(disposition)
}

func (h *Handler) addIgnoredSimilarCollectorRule(ctx context.Context, fingerprint string, event ignoredSimilarEvent) error {
	rule := ignoredSimilarCollectorRule(fingerprint, event)
	if len(rule.Conditions) == 0 {
		return nil
	}
	config, err := h.collectorFilterRepository.GetCollectorFilter(ctx)
	if err != nil {
		return err
	}
	config.Enabled = true
	config.Rules = upsertCollectorFilterRule(config.Rules, rule)
	return h.collectorFilterRepository.SaveCollectorFilter(ctx, config)
}

type ignoredSimilarEvent struct {
	EventType     string   `json:"eventType"`
	Severity      string   `json:"severity"`
	Username      string   `json:"username"`
	LoginUsername string   `json:"loginUsername"`
	ProcessName   string   `json:"processName"`
	Cmdline       string   `json:"cmdline"`
	FilePath      string   `json:"filePath"`
	FileOperation string   `json:"fileOperation"`
	DstIP         string   `json:"dstIp"`
	DstPort       uint16   `json:"dstPort"`
	Protocol      string   `json:"protocol"`
	RuleIDs       []string `json:"ruleIds"`
}

func ignoredSimilarCollectorRule(fingerprint string, event ignoredSimilarEvent) systemconfig.CollectorFilterRule {
	idSource := strings.TrimSpace(fingerprint)
	if idSource == "" {
		idSource = strings.TrimSpace(event.EventType + "|" + event.ProcessName + "|" + event.Cmdline + "|" + event.FilePath)
	}
	rule := systemconfig.CollectorFilterRule{
		ID:      "risk-ignore-similar-" + stableRuleID(idSource),
		Name:    "风险处置忽略同类",
		Enabled: true,
	}
	addCondition := func(field, op, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		rule.Conditions = append(rule.Conditions, systemconfig.CollectorFilterCondition{Field: field, Op: op, Value: value})
	}
	addCondition("event_type", "eq", event.EventType)
	addCondition("process_name", "eq", event.ProcessName)
	if op, value := similarCommandCondition(event); value != "" {
		addCondition("cmdline", op, value)
	}
	addCondition("file_path", "eq", event.FilePath)
	addCondition("file_operation", "eq", event.FileOperation)
	addCondition("dst_ip", "eq", event.DstIP)
	if event.DstPort != 0 {
		addCondition("dst_port", "eq", fmt.Sprintf("%d", event.DstPort))
	}
	addCondition("protocol", "eq", event.Protocol)
	return rule
}

func similarCommandCondition(event ignoredSimilarEvent) (string, string) {
	cmdline := strings.TrimSpace(event.Cmdline)
	if cmdline == "" {
		return "", ""
	}
	if strings.EqualFold(strings.TrimSpace(event.ProcessName), "runc") && strings.Contains(cmdline, " exec ") {
		return "regex", `(^|.*/)\brunc\b.*\bexec\b`
	}
	return "eq", cmdline
}

func stableRuleID(value string) string {
	replacer := strings.NewReplacer(" ", "-", "|", "-", "/", "-", "\\", "-", ":", "-", ".", "-", "_", "-")
	id := strings.ToLower(replacer.Replace(value))
	id = strings.Trim(id, "-")
	if len(id) > 48 {
		return id[:48]
	}
	if id == "" {
		return "event"
	}
	return id
}

func upsertCollectorFilterRule(rules []systemconfig.CollectorFilterRule, rule systemconfig.CollectorFilterRule) []systemconfig.CollectorFilterRule {
	for index := range rules {
		if rules[index].ID == rule.ID {
			rules[index] = rule
			return rules
		}
	}
	return append(rules, rule)
}
