package systemconfig

import (
	"context"
	"fmt"
	"sync"
)

const CollectorFilterKey = "collector_filter"

type CollectorFilterConfig struct {
	Enabled               bool                  `json:"enabled"`
	IgnoreProcessNames    []string              `json:"ignoreProcessNames"`
	IgnoreCommandKeywords []string              `json:"ignoreCommandKeywords"`
	IgnoreUsers           []string              `json:"ignoreUsers"`
	KeepSeverities        []string              `json:"keepSeverities"`
	Rules                 []CollectorFilterRule `json:"rules"`
}

type CollectorFilterRule struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Enabled    bool                       `json:"enabled"`
	Conditions []CollectorFilterCondition `json:"conditions"`
}

type CollectorFilterCondition struct {
	Field  string   `json:"field"`
	Op     string   `json:"op"`
	Value  string   `json:"value"`
	Values []string `json:"values"`
}

type Repository interface {
	GetCollectorFilter(ctx context.Context) (CollectorFilterConfig, error)
	SaveCollectorFilter(ctx context.Context, config CollectorFilterConfig) error
}

// DefaultCollectorFilterConfig 处理 Default Collector Filter Config 相关逻辑。
func DefaultCollectorFilterConfig() CollectorFilterConfig {
	return CollectorFilterConfig{
		Enabled:        false,
		KeepSeverities: []string{"high", "critical"},
	}
}

// PreReleaseCollectorFilterConfig returns a practical baseline for pre-release data collection.
func PreReleaseCollectorFilterConfig() CollectorFilterConfig {
	return normalizeCollectorFilterConfig(CollectorFilterConfig{
		Enabled:        true,
		KeepSeverities: []string{"high", "critical"},
		Rules: []CollectorFilterRule{
			{
				ID:      "pre-root-process-low-risk",
				Name:    "预发忽略 root 常规命令",
				Enabled: true,
				Conditions: []CollectorFilterCondition{
					{Field: "event_type", Op: "eq", Value: "process_exec"},
					{Field: "username", Op: "eq", Value: "root"},
					{Field: "severity", Op: "in", Values: []string{"info", "low", "medium"}},
				},
			},
			{
				ID:      "pre-root-file-low-risk",
				Name:    "预发忽略 root 常规文件访问",
				Enabled: true,
				Conditions: []CollectorFilterCondition{
					{Field: "event_type", Op: "eq", Value: "file_access"},
					{Field: "username", Op: "eq", Value: "root"},
					{Field: "severity", Op: "in", Values: []string{"info", "low", "medium"}},
				},
			},
			{
				ID:      "pre-root-network-low-risk",
				Name:    "预发忽略 root 低风险网络连接",
				Enabled: true,
				Conditions: []CollectorFilterCondition{
					{Field: "event_type", Op: "eq", Value: "network_connect"},
					{Field: "username", Op: "eq", Value: "root"},
					{Field: "severity", Op: "in", Values: []string{"info", "low", "medium"}},
				},
			},
			{
				ID:      "pre-proc-sys-read-noise",
				Name:    "预发忽略 proc/sys/dev 高频读取",
				Enabled: true,
				Conditions: []CollectorFilterCondition{
					{Field: "event_type", Op: "eq", Value: "file_access"},
					{Field: "file_path", Op: "regex", Value: "^/(proc|sys)/|^/dev/(null|zero|random|urandom)$"},
					{Field: "file_operation", Op: "regex", Value: "(?i)(open|read|security_file_open|security_file_permission)"},
					{Field: "severity", Op: "in", Values: []string{"info", "low", "medium"}},
				},
			},
			{
				ID:      "pre-monitoring-agent-noise",
				Name:    "预发忽略监控探针噪声",
				Enabled: true,
				Conditions: []CollectorFilterCondition{
					{Field: "process_name", Op: "in", Values: []string{"kube-probe", "node_exporter", "prometheus", "telegraf", "grafana-agent"}},
					{Field: "severity", Op: "in", Values: []string{"info", "low", "medium"}},
				},
			},
		},
	})
}

type MemoryRepository struct {
	mu              sync.Mutex
	collectorFilter CollectorFilterConfig
	hasFilter       bool
}

// NewMemoryRepository 创建并初始化 New Memory Repository 实例。
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

// GetCollectorFilter 查询并返回指定的 Get Collector Filter。
func (r *MemoryRepository) GetCollectorFilter(_ context.Context) (CollectorFilterConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.hasFilter {
		return DefaultCollectorFilterConfig(), nil
	}
	return normalizeCollectorFilterConfig(r.collectorFilter), nil
}

// SaveCollectorFilter 处理 Save Collector Filter 相关逻辑。
func (r *MemoryRepository) SaveCollectorFilter(_ context.Context, config CollectorFilterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectorFilter = normalizeCollectorFilterConfig(config)
	r.hasFilter = true
	return nil
}

// normalizeCollectorFilterConfig 规范化 normalize Collector Filter Config 的默认值和边界值。
func normalizeCollectorFilterConfig(config CollectorFilterConfig) CollectorFilterConfig {
	if len(config.KeepSeverities) == 0 {
		config.KeepSeverities = []string{"high", "critical"}
	}
	if config.IgnoreProcessNames == nil {
		config.IgnoreProcessNames = []string{}
	}
	if config.IgnoreCommandKeywords == nil {
		config.IgnoreCommandKeywords = []string{}
	}
	if config.IgnoreUsers == nil {
		config.IgnoreUsers = []string{}
	}
	if config.Rules == nil {
		config.Rules = []CollectorFilterRule{}
	}
	if len(config.Rules) == 0 {
		config.Rules = legacyCollectorFilterRules(config)
	}
	for index := range config.Rules {
		if config.Rules[index].Conditions == nil {
			config.Rules[index].Conditions = []CollectorFilterCondition{}
		}
		for conditionIndex := range config.Rules[index].Conditions {
			if config.Rules[index].Conditions[conditionIndex].Values == nil {
				config.Rules[index].Conditions[conditionIndex].Values = []string{}
			}
		}
	}
	return config
}

func legacyCollectorFilterRules(config CollectorFilterConfig) []CollectorFilterRule {
	rules := []CollectorFilterRule{}
	for index, processName := range config.IgnoreProcessNames {
		if processName == "" {
			continue
		}
		rules = append(rules, CollectorFilterRule{
			ID:      fmt.Sprintf("legacy-process-%d", index),
			Name:    "忽略进程 " + processName,
			Enabled: true,
			Conditions: []CollectorFilterCondition{
				{Field: "process_name", Op: "eq", Value: processName},
			},
		})
	}
	for index, keyword := range config.IgnoreCommandKeywords {
		if keyword == "" {
			continue
		}
		rules = append(rules, CollectorFilterRule{
			ID:      fmt.Sprintf("legacy-command-%d", index),
			Name:    "忽略命令 " + keyword,
			Enabled: true,
			Conditions: []CollectorFilterCondition{
				{Field: "cmdline", Op: "contains", Value: keyword},
			},
		})
	}
	for index, username := range config.IgnoreUsers {
		if username == "" {
			continue
		}
		rules = append(rules, CollectorFilterRule{
			ID:      fmt.Sprintf("legacy-user-%d", index),
			Name:    "忽略执行用户 " + username,
			Enabled: true,
			Conditions: []CollectorFilterCondition{
				{Field: "username", Op: "eq", Value: username},
			},
		})
		rules = append(rules, CollectorFilterRule{
			ID:      fmt.Sprintf("legacy-login-user-%d", index),
			Name:    "忽略登录用户 " + username,
			Enabled: true,
			Conditions: []CollectorFilterCondition{
				{Field: "login_username", Op: "eq", Value: username},
			},
		})
	}
	return rules
}
