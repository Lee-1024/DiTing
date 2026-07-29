package systemconfig

import (
	"context"
	"testing"
)

func TestMemoryRepositoryReturnsDefaultCollectorFilterConfig(t *testing.T) {
	repository := NewMemoryRepository()

	config, err := repository.GetCollectorFilter(context.Background())
	if err != nil {
		t.Fatalf("GetCollectorFilter returned error: %v", err)
	}

	if config.Enabled {
		t.Fatalf("expected collector filter to be disabled by default")
	}
	if len(config.KeepSeverities) != 2 || config.KeepSeverities[0] != "high" || config.KeepSeverities[1] != "critical" {
		t.Fatalf("unexpected default keep severities: %#v", config.KeepSeverities)
	}
}

func TestMemoryRepositorySavesCollectorFilterConfig(t *testing.T) {
	repository := NewMemoryRepository()
	expected := CollectorFilterConfig{
		Enabled:               true,
		IgnoreProcessNames:    []string{"node_exporter"},
		IgnoreCommandKeywords: []string{"/metrics"},
		IgnoreUsers:           []string{"prometheus"},
		KeepSeverities:        []string{"high", "critical"},
		Rules: []CollectorFilterRule{{
			ID:      "rule-1",
			Name:    "ignore vite",
			Enabled: true,
			Conditions: []CollectorFilterCondition{
				{Field: "process_name", Op: "eq", Value: "node"},
				{Field: "cmdline", Op: "contains", Value: "node_modules/.bin/vite"},
			},
		}},
	}

	if err := repository.SaveCollectorFilter(context.Background(), expected); err != nil {
		t.Fatalf("SaveCollectorFilter returned error: %v", err)
	}
	actual, err := repository.GetCollectorFilter(context.Background())
	if err != nil {
		t.Fatalf("GetCollectorFilter returned error: %v", err)
	}

	if !actual.Enabled {
		t.Fatalf("expected collector filter to be enabled")
	}
	if actual.IgnoreProcessNames[0] != "node_exporter" {
		t.Fatalf("unexpected ignored process names: %#v", actual.IgnoreProcessNames)
	}
	if actual.IgnoreCommandKeywords[0] != "/metrics" {
		t.Fatalf("unexpected ignored command keywords: %#v", actual.IgnoreCommandKeywords)
	}
	if actual.IgnoreUsers[0] != "prometheus" {
		t.Fatalf("unexpected ignored users: %#v", actual.IgnoreUsers)
	}
	if len(actual.Rules) != 1 || actual.Rules[0].Conditions[1].Value != "node_modules/.bin/vite" {
		t.Fatalf("unexpected rules: %#v", actual.Rules)
	}
}
