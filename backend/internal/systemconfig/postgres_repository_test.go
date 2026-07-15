package systemconfig

import (
	"encoding/json"
	"testing"
)

func TestCollectorFilterJSONRoundTripForPostgres(t *testing.T) {
	config := CollectorFilterConfig{
		Enabled:               true,
		IgnoreProcessNames:    []string{"node_exporter"},
		IgnoreCommandKeywords: []string{"/metrics"},
		IgnoreUsers:           []string{"prometheus"},
		KeepSeverities:        []string{"high", "critical"},
	}

	data, err := marshalCollectorFilterConfig(config)
	if err != nil {
		t.Fatalf("marshalCollectorFilterConfig returned error: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw json: %v", err)
	}
	if raw["enabled"] != true {
		t.Fatalf("expected enabled true, got %#v", raw["enabled"])
	}

	decoded, err := unmarshalCollectorFilterConfig(data)
	if err != nil {
		t.Fatalf("unmarshalCollectorFilterConfig returned error: %v", err)
	}
	if !decoded.Enabled || decoded.IgnoreUsers[0] != "prometheus" || decoded.KeepSeverities[1] != "critical" {
		t.Fatalf("unexpected decoded config: %#v", decoded)
	}
}
