package enforcement

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerCreatesAndListsPolicies(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	body := bytes.NewBufferString(`{"name":"保护敏感文件","template":"sensitive_file","mode":"enforce","enabled":true,"targetHosts":["host-1"],"yaml":"kind: TracingPolicy"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/enforcement-policies", body)
	createResp := httptest.NewRecorder()

	handler.Create(createResp, createReq)

	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createResp.Code, createResp.Body.String())
	}
	var created Policy
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created policy: %v", err)
	}
	if created.Name != "保护敏感文件" || created.DeploymentStatus != "draft" {
		t.Fatalf("unexpected created policy: %#v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/enforcement-policies", nil)
	listResp := httptest.NewRecorder()
	handler.List(listResp, listReq)

	if listResp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", listResp.Code)
	}
	var policies []Policy
	if err := json.NewDecoder(listResp.Body).Decode(&policies); err != nil {
		t.Fatalf("decode policies: %v", err)
	}
	if len(policies) != 1 || policies[0].ID != created.ID {
		t.Fatalf("expected created policy in list, got %#v", policies)
	}
}

func TestHandlerRejectsPolicyWithoutYAML(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enforcement-policies", bytes.NewBufferString(`{"name":"bad","template":"sensitive_file"}`))
	resp := httptest.NewRecorder()

	handler.Create(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestHandlerUpdatesDeploymentStatus(t *testing.T) {
	repository := NewMemoryRepository()
	created, err := repository.Create(nil, Policy{Name: "策略", Template: "delete_behavior", YAML: "kind: TracingPolicy"})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	handler := NewHandler(repository)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enforcement-policies/"+created.ID+"/deployment", bytes.NewBufferString(`{"status":"deployed","message":"已放入策略目录"}`))
	req.SetPathValue("id", created.ID)
	resp := httptest.NewRecorder()

	handler.UpdateDeployment(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var updated Policy
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated policy: %v", err)
	}
	if updated.DeploymentStatus != "deployed" || updated.DeployedAt == nil {
		t.Fatalf("expected deployed status with deployedAt, got %#v", updated)
	}
}
