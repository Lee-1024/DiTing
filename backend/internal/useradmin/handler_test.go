package useradmin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerCreatesAndListsUsers(t *testing.T) {
	repository := NewMemoryRepository()
	handler := NewHandler(repository)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"username":"operator","password":"secret123","displayName":"Operator","email":"operator@example.com","status":"active","roles":["admin"]}`))
	handler.CreateUser(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	if strings.Contains(createRec.Body.String(), "secret123") {
		t.Fatalf("expected password to be omitted from response, got %s", createRec.Body.String())
	}

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	handler.ListUsers(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", listRec.Code)
	}
	if !strings.Contains(listRec.Body.String(), `"username":"operator"`) {
		t.Fatalf("expected created user in list, got %s", listRec.Body.String())
	}
}

func TestHandlerRejectsWeakPassword(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{"username":"operator","password":"123","displayName":"Operator"}`))

	handler.CreateUser(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestHandlerUpdatesUserAndResetsPassword(t *testing.T) {
	repository := NewMemoryRepository()
	created, err := repository.CreateUser(nil, CreateUserRequest{Username: "operator", Password: "secret123", DisplayName: "Operator", Status: "active", Roles: []string{"admin"}})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	handler := NewHandler(repository)

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+created.ID, bytes.NewBufferString(`{"displayName":"Ops","email":"ops@example.com","status":"active","roles":["admin"]}`))
	updateReq.SetPathValue("id", created.ID)
	handler.UpdateUser(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	if !strings.Contains(updateRec.Body.String(), `"displayName":"Ops"`) {
		t.Fatalf("expected updated user, got %s", updateRec.Body.String())
	}

	resetRec := httptest.NewRecorder()
	resetReq := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+created.ID+"/password", bytes.NewBufferString(`{"password":"newpass123"}`))
	resetReq.SetPathValue("id", created.ID)
	handler.ResetPassword(resetRec, resetReq)

	if resetRec.Code != http.StatusNoContent {
		t.Fatalf("expected reset status 204, got %d: %s", resetRec.Code, resetRec.Body.String())
	}
}

func TestHandlerPreventsDeletingLastAdmin(t *testing.T) {
	repository := NewMemoryRepository()
	created, err := repository.CreateUser(nil, CreateUserRequest{Username: "admin", Password: "secret123", DisplayName: "Admin", Status: "active", Roles: []string{"admin"}})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	handler := NewHandler(repository)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+created.ID, nil)
	req.SetPathValue("id", created.ID)
	handler.DeleteUser(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
}

func TestHandlerListsRoles(t *testing.T) {
	handler := NewHandler(NewMemoryRepository())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles", nil)

	handler.ListRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"name":"admin"`) {
		t.Fatalf("expected admin role, got %s", rec.Body.String())
	}
}
