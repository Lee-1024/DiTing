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
}
