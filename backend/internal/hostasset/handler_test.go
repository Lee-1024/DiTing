package hostasset

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHostAssetHandlerCreatesAndListsAssets(t *testing.T) {
	repository := NewMemoryRepository()
	handler := NewHandler(repository)

	body := bytes.NewBufferString(`{"nodeName":"dd9f5f94c8e2","displayName":"prod-web-01","hostIp":"10.0.0.1","environment":"prod","owner":"ops"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/host-assets", body)
	createRec := httptest.NewRecorder()
	handler.Create(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRec.Code, createRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/host-assets", nil)
	listRec := httptest.NewRecorder()
	handler.List(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), `"displayName":"prod-web-01"`) {
		t.Fatalf("unexpected body %s", listRec.Body.String())
	}
}
