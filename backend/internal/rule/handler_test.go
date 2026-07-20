package rule

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRuleValidatesRequiredName(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"eventType":"process_exec","severity":"high","riskScore":80,"matchExpr":{"operator":"and","conditions":[]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", body)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateRuleValidatesSeverity(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"name":"bad","eventType":"process_exec","severity":"urgent","riskScore":80,"matchExpr":{"operator":"and","conditions":[]}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", body)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateAndListRules(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"name":"reverse shell","eventType":"process_exec","severity":"high","riskScore":80,"matchExpr":{"operator":"and","conditions":[{"field":"cmdline","op":"contains","value":"bash -i"}]},"tags":["reverse-shell"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", body)
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)
	listRec := httptest.NewRecorder()
	handler.List(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRec.Code)
	}

	var response []Rule
	if err := json.Unmarshal(listRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected one rule, got %d", len(response))
	}
	if response[0].Name != "reverse shell" {
		t.Fatalf("unexpected rule name %q", response[0].Name)
	}
}

func TestListRulesReturnsEmptyArray(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)
	rec := httptest.NewRecorder()

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "[]\n" {
		t.Fatalf("expected empty array, got %q", rec.Body.String())
	}
}

func TestTestRuleReturnsMatchedConditions(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"rule":{"name":"反弹 Shell 命令","eventType":"process_exec","enabled":true,"severity":"critical","riskScore":95,"matchExpr":{"operator":"and","conditions":[{"field":"cmdline","op":"contains","value":"bash -i"},{"field":"username","op":"eq","value":"ubuntu"}]}},"event":{"eventType":"process_exec","cmdline":"/bin/bash -i","username":"ubuntu"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/test", body)
	rec := httptest.NewRecorder()

	handler.Test(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response TestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Matched {
		t.Fatal("expected rule to match")
	}
	if len(response.Matches) != 2 {
		t.Fatalf("expected two matched conditions, got %d", len(response.Matches))
	}
	if response.Matches[0].Field != "cmdline" || response.Matches[0].Actual != "/bin/bash -i" {
		t.Fatalf("unexpected first match: %#v", response.Matches[0])
	}
}

func TestTestRuleMatchesNetworkFields(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"rule":{"name":"网络外联","eventType":"network_connect","enabled":true,"severity":"high","riskScore":80,"matchExpr":{"operator":"and","conditions":[{"field":"event_type","op":"eq","value":"network_connect"},{"field":"dst_ip","op":"eq","value":"10.0.0.8"},{"field":"dst_port","op":"eq","value":"443"},{"field":"protocol","op":"eq","value":"tcp"}]}},"event":{"eventType":"network_connect","dstIp":"10.0.0.8","dstPort":443,"protocol":"tcp"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/test", body)
	rec := httptest.NewRecorder()

	handler.Test(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response TestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Matched {
		t.Fatalf("expected network rule to match, got %#v", response)
	}
}

func TestTestRuleReturnsUnmatchedReason(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"rule":{"name":"docker","eventType":"process_exec","enabled":true,"severity":"high","riskScore":70,"matchExpr":{"operator":"and","conditions":[{"field":"process_name","op":"eq","value":"docker"}]}},"event":{"eventType":"process_exec","processName":"bash","cmdline":"id"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules/test", body)
	rec := httptest.NewRecorder()

	handler.Test(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response TestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Matched {
		t.Fatal("expected rule not to match")
	}
	if response.Message == "" {
		t.Fatal("expected unmatched message")
	}
}

func TestUpdateAndDeleteRule(t *testing.T) {
	repository := NewMemoryRepository()
	handler := NewHandler(repository)
	created, err := repository.Create(t.Context(), Rule{
		Name:      "reverse shell",
		EventType: "process_exec",
		Enabled:   true,
		Severity:  "high",
		RiskScore: 85,
		MatchExpr: Expression{Operator: "and", Conditions: []Condition{{Field: "cmdline", Op: "contains", Value: "bash -i"}}},
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	updateBody := bytes.NewBufferString(`{"name":"reverse shell updated","eventType":"process_exec","enabled":false,"severity":"critical","riskScore":95,"matchExpr":{"operator":"and","conditions":[{"field":"cmdline","op":"contains","value":"nc -e"}]},"tags":["critical-command"]}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/rules/"+created.ID, updateBody)
	updateReq.SetPathValue("id", created.ID)
	updateRec := httptest.NewRecorder()
	handler.Update(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	updated, err := repository.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("get updated rule: %v", err)
	}
	if updated.Severity != "critical" || updated.Enabled {
		t.Fatalf("expected critical disabled rule, got severity=%q enabled=%v", updated.Severity, updated.Enabled)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/rules/"+created.ID, nil)
	deleteReq.SetPathValue("id", created.ID)
	deleteRec := httptest.NewRecorder()
	handler.Delete(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", deleteRec.Code)
	}
	rules, err := repository.List(t.Context())
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("expected no rules after delete, got %d", len(rules))
	}
}
