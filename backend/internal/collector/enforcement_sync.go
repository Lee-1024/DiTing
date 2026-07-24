package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type EnforcementPolicy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

type EnforcementSyncer struct {
	baseURL        string
	token          string
	hostID         string
	hostName       string
	policyDir      string
	restartCommand string
	client         *http.Client
	runCommand     func(context.Context, string) error
}

// NewEnforcementSyncer 创建并初始化 New Enforcement Syncer 实例。
func NewEnforcementSyncer(ingestURL string, token string, hostID string, hostName string, policyDir string, restartCommand string) *EnforcementSyncer {
	return &EnforcementSyncer{
		baseURL:        enforcementBaseURL(ingestURL),
		token:          strings.TrimSpace(token),
		hostID:         strings.TrimSpace(hostID),
		hostName:       strings.TrimSpace(hostName),
		policyDir:      strings.TrimSpace(policyDir),
		restartCommand: strings.TrimSpace(restartCommand),
		client:         &http.Client{Timeout: 30 * time.Second},
		runCommand:     runShellCommand,
	}
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
	changed, err := s.applyPolicies(policies)
	if err != nil {
		return err
	}
	status := "deployed"
	message := "策略已同步"
	if changed && s.restartCommand != "" {
		if err := s.runCommand(ctx, s.restartCommand); err != nil {
			status = "failed"
			message = "策略已写入但重启 Tetragon 失败: " + err.Error()
		} else {
			message = "策略已同步并重启 Tetragon"
		}
	}
	for _, policy := range policies {
		_ = s.reportDeployment(ctx, policy.ID, status, message)
	}
	return nil
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

// applyPolicies 处理 apply Policies 相关逻辑。
func (s *EnforcementSyncer) applyPolicies(policies []EnforcementPolicy) (bool, error) {
	if err := os.MkdirAll(s.policyDir, 0o755); err != nil {
		return false, err
	}
	desired := map[string]string{}
	for _, policy := range policies {
		desired[policyFileName(policy)] = policy.YAML
	}
	changed := false
	for fileName, content := range desired {
		path := filepath.Join(s.policyDir, fileName)
		current, err := os.ReadFile(path)
		if err == nil && string(current) == content {
			continue
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return changed, err
		}
		changed = true
	}
	files, err := filepath.Glob(filepath.Join(s.policyDir, "diting-*.yaml"))
	if err != nil {
		return changed, err
	}
	for _, path := range files {
		if _, ok := desired[filepath.Base(path)]; ok {
			continue
		}
		if err := os.Remove(path); err != nil {
			return changed, err
		}
		changed = true
	}
	return changed, nil
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

// sanitizePolicyFileName 处理 sanitize Policy File Name 相关逻辑。
func sanitizePolicyFileName(value string) string {
	name := strings.ToLower(value)
	var builder strings.Builder
	lastDash := false
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "diting-policy"
	}
	if !strings.HasPrefix(result, "diting-") {
		return "diting-" + result
	}
	return result
}

// policyFileName 处理 policy File Name 相关逻辑。
func policyFileName(policy EnforcementPolicy) string {
	name := sanitizePolicyFileName(policy.Name)
	id := sanitizePolicyFileName(policy.ID)
	if id == "diting-policy" {
		return name + ".yaml"
	}
	return name + "-" + strings.TrimPrefix(id, "diting-") + ".yaml"
}

// runShellCommand 运行 run Shell Command 的主流程。
func runShellCommand(ctx context.Context, command string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
