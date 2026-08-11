package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type EnforcementPolicy struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Template   string          `json:"template"`
	Mode       string          `json:"mode"`
	Enabled    bool            `json:"enabled"`
	Definition json.RawMessage `json:"definition"`
	YAML       string          `json:"yaml"`
}

type appArmorDeploymentResult struct {
	Status  string
	Message string
}

type sensitiveFileDefinition struct {
	FilePaths     []string `json:"filePaths"`
	Operations    []string `json:"operations"`
	UserMatchMode string   `json:"userMatchMode"`
}

func buildAppArmorDeployment(policies []EnforcementPolicy) (string, string, map[string]appArmorDeploymentResult) {
	results := make(map[string]appArmorDeploymentResult, len(policies))
	var protectedPaths []string
	var appArmorRules []AppArmorPathRule
	for _, policy := range policies {
		if !policy.Enabled || policy.Mode == "disabled" {
			results[policy.ID] = appArmorDeploymentResult{Status: "disabled", Message: "策略已停用"}
			continue
		}
		if policy.Template != "sensitive_file" || policy.Mode != "enforce" {
			results[policy.ID] = appArmorDeploymentResult{Status: "failed", Message: "当前 AppArmor 首版不支持该拦截模板"}
			continue
		}
		var definition sensitiveFileDefinition
		if err := json.Unmarshal(policy.Definition, &definition); err != nil {
			results[policy.ID] = appArmorDeploymentResult{Status: "failed", Message: "策略 definition 无效: " + err.Error()}
			continue
		}
		if _, err := normalizeAppArmorOperations(definition.Operations); err != nil {
			results[policy.ID] = appArmorDeploymentResult{Status: "failed", Message: "策略 operations 无效: " + err.Error()}
			continue
		}
		paths, err := normalizeAppArmorPaths(definition.FilePaths)
		if err != nil {
			results[policy.ID] = appArmorDeploymentResult{Status: "failed", Message: err.Error()}
			continue
		}
		protectedPaths = append(protectedPaths, paths...)
		for _, path := range paths {
			appArmorRules = append(appArmorRules, AppArmorPathRule{Path: path, Operations: definition.Operations})
		}
		results[policy.ID] = appArmorDeploymentResult{Status: "deployed", Message: "AppArmor 策略已加载，保护操作: " + strings.Join(normalizeSensitiveFileOperations(definition.Operations), ", ")}
	}
	if len(appArmorRules) == 0 {
		return "", "", results
	}
	profile, err := GenerateAppArmorSudoProfile(appArmorRules)
	if err != nil {
		for id, result := range results {
			if result.Status == "deployed" {
				results[id] = appArmorDeploymentResult{Status: "failed", Message: err.Error()}
			}
		}
		return "", "", results
	}
	observerPolicy, err := GenerateTetragonObserverPolicy(protectedPaths)
	if err != nil {
		return "", "", results
	}
	return profile, observerPolicy, results
}

func normalizeSensitiveFileOperations(operations []string) []string {
	if len(operations) == 0 {
		return []string{"write"}
	}
	seen := make(map[string]struct{}, len(operations))
	result := make([]string, 0, len(operations))
	for _, operation := range operations {
		operation = strings.TrimSpace(strings.ToLower(operation))
		if operation == "" {
			continue
		}
		if _, ok := seen[operation]; ok {
			continue
		}
		seen[operation] = struct{}{}
		result = append(result, operation)
	}
	return result
}

type EnforcementSyncer struct {
	baseURL       string
	token         string
	hostID        string
	hostName      string
	policyDir     string
	client        *http.Client
	appArmor      appArmorProfileManager
	observer      tetragonObserverPolicyManager
	capabilityErr error
}

type tetragonObserverPolicyManager interface {
	Apply(context.Context, string) error
	Remove(context.Context) error
}

type appArmorProfileManager interface {
	Apply(context.Context, string) (bool, error)
	Remove(context.Context) (bool, error)
}

// NewEnforcementSyncer 创建并初始化 New Enforcement Syncer 实例。
func NewEnforcementSyncer(ingestURL string, token string, hostID string, hostName string, policyDir string, tetragonGRPCAddr string) *EnforcementSyncer {
	syncer := &EnforcementSyncer{
		baseURL:   enforcementBaseURL(ingestURL),
		token:     strings.TrimSpace(token),
		hostID:    strings.TrimSpace(hostID),
		hostName:  strings.TrimSpace(hostName),
		policyDir: strings.TrimSpace(policyDir),
		client:    &http.Client{Timeout: 30 * time.Second},
	}
	manager, err := discoverAppArmorManager(syncer.policyDir)
	if err != nil {
		syncer.capabilityErr = err
		return syncer
	}
	syncer.appArmor = manager
	observer, observerErr := NewTetragonObserverManager(tetragonGRPCAddr)
	if observerErr != nil {
		syncer.capabilityErr = observerErr
		return syncer
	}
	syncer.observer = observer
	return syncer
}

// SyncOnce 处理 Sync Once 相关逻辑。
func (s *EnforcementSyncer) SyncOnce(ctx context.Context) error {
	if s.hostID == "" {
		return fmt.Errorf("host id is required")
	}
	if s.policyDir == "" {
		return fmt.Errorf("enforcement policy dir is required")
	}
	policies, err := s.fetchPolicies(ctx)
	if err != nil {
		return err
	}
	profile, observerPolicy, results := buildAppArmorDeployment(policies)
	var applyErr error
	if s.appArmor == nil {
		applyErr = s.capabilityErr
		if applyErr == nil {
			applyErr = fmt.Errorf("AppArmor manager is unavailable")
		}
	} else if profile == "" {
		_, applyErr = s.appArmor.Remove(ctx)
		if applyErr == nil && s.observer != nil {
			applyErr = s.observer.Remove(ctx)
		}
	} else {
		_, applyErr = s.appArmor.Apply(ctx, profile)
		if applyErr == nil {
			if s.observer == nil {
				applyErr = fmt.Errorf("Tetragon observer manager is unavailable")
			} else {
				applyErr = s.observer.Apply(ctx, observerPolicy)
			}
		}
	}
	if applyErr != nil {
		for id, result := range results {
			if result.Status == "deployed" {
				results[id] = appArmorDeploymentResult{Status: "failed", Message: "拦截或观测策略加载失败: " + applyErr.Error()}
			}
		}
	}
	for _, policy := range policies {
		result := results[policy.ID]
		_ = s.reportDeployment(ctx, policy.ID, result.Status, result.Message)
	}
	return applyErr
}

// Run 运行 Run 的主流程。
func (s *EnforcementSyncer) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		_ = s.SyncOnce(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// fetchPolicies 处理 fetch Policies 相关逻辑。
func (s *EnforcementSyncer) fetchPolicies(ctx context.Context) ([]EnforcementPolicy, error) {
	url := fmt.Sprintf("%s/ingest/enforcement-policies?host_id=%s", s.baseURL, s.hostID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch enforcement policies status %d", resp.StatusCode)
	}
	var policies []EnforcementPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policies); err != nil {
		return nil, err
	}
	return policies, nil
}

// reportDeployment 处理 report Deployment 相关逻辑。
func (s *EnforcementSyncer) reportDeployment(ctx context.Context, policyID string, status string, message string) error {
	payload := map[string]string{
		"hostId":   s.hostID,
		"hostName": s.hostName,
		"status":   status,
		"message":  message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/ingest/enforcement-policies/%s/deployments", s.baseURL, policyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("report enforcement deployment status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return nil
}

// enforcementBaseURL 处理 enforcement Base URL 相关逻辑。
func enforcementBaseURL(ingestURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(ingestURL), "/")
	if strings.HasSuffix(trimmed, "/ingest/events") {
		return strings.TrimSuffix(trimmed, "/ingest/events")
	}
	if strings.HasSuffix(trimmed, "/events") {
		return strings.TrimSuffix(trimmed, "/events")
	}
	return trimmed
}
