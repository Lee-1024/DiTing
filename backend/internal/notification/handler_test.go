package notification

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"diting/backend/internal/auth"
)

func notificationRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	return req.WithContext(auth.ContextWithClaims(req.Context(), auth.Claims{UserID: "00000000-0000-0000-0000-000000000001", Username: "admin"}))
}

func TestHandlerListsNotificationsForCurrentUser(t *testing.T) {
	repo := NewMemoryRepository()
	_, _ = repo.Upsert(context.Background(), Input{Type: TypeCollector, DedupeKey: "collector:offline:host-1", Title: "采集节点离线"})
	handler := NewHandler(repo)
	recorder := httptest.NewRecorder()

	handler.List(recorder, notificationRequest(http.MethodGet, "/api/v1/notifications?view=unread", ""))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"unread":1`) {
		t.Fatalf("unexpected response code=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestHandlerMarksNotificationRead(t *testing.T) {
	repo := NewMemoryRepository()
	item, _ := repo.Upsert(context.Background(), Input{Type: TypeCollector, DedupeKey: "collector:offline:host-1", Title: "采集节点离线"})
	handler := NewHandler(repo)
	recorder := httptest.NewRecorder()
	req := notificationRequest(http.MethodPost, "/api/v1/notifications/"+item.ID+"/read", "")
	req.SetPathValue("id", item.ID)

	handler.MarkRead(recorder, req)

	result, _ := repo.List(context.Background(), "00000000-0000-0000-0000-000000000001", "unread", 20)
	if recorder.Code != http.StatusNoContent || result.Counts.Unread != 0 {
		t.Fatalf("notification was not marked read: code=%d result=%#v", recorder.Code, result)
	}
}

func TestHandlerMarksAllNotificationsRead(t *testing.T) {
	repo := NewMemoryRepository()
	_, _ = repo.Upsert(context.Background(), Input{Type: TypeCollector, DedupeKey: "collector:offline:host-1", Title: "采集节点离线"})
	_, _ = repo.Upsert(context.Background(), Input{Type: TypeCollector, DedupeKey: "collector:offline:host-2", Title: "采集节点离线"})
	handler := NewHandler(repo)
	recorder := httptest.NewRecorder()

	handler.MarkAllRead(recorder, notificationRequest(http.MethodPost, "/api/v1/notifications/read-all", ""))

	result, _ := repo.List(context.Background(), "00000000-0000-0000-0000-000000000001", "unread", 20)
	if recorder.Code != http.StatusNoContent || result.Counts.Unread != 0 {
		t.Fatalf("notifications were not marked read: code=%d result=%#v", recorder.Code, result)
	}
}

func TestHandlerDisposesEnforcementNotification(t *testing.T) {
	repo := NewMemoryRepository()
	item, _ := repo.Upsert(context.Background(), Input{Type: TypeEnforcement, DedupeKey: "enforcement:event-1", SourceID: "event-1", Title: "拦截策略触发"})
	handler := NewHandler(repo)
	recorder := httptest.NewRecorder()
	req := notificationRequest(http.MethodPost, "/api/v1/notifications/"+item.ID+"/handle", `{"disposition":"confirmed"}`)
	req.SetPathValue("id", item.ID)

	handler.Handle(recorder, req)

	pending, _ := repo.List(context.Background(), "00000000-0000-0000-0000-000000000001", "pending", 20)
	unread, _ := repo.List(context.Background(), "00000000-0000-0000-0000-000000000001", "unread", 20)
	if recorder.Code != http.StatusNoContent || len(pending.Items) != 0 || unread.Counts.Unread != 0 {
		t.Fatalf("notification was not disposed: code=%d pending=%#v unread=%#v", recorder.Code, pending, unread)
	}
}

func TestHandlerRejectsInvalidDisposition(t *testing.T) {
	repo := NewMemoryRepository()
	item, _ := repo.Upsert(context.Background(), Input{Type: TypeEnforcement, DedupeKey: "enforcement:event-1", SourceID: "event-1", Title: "拦截策略触发"})
	handler := NewHandler(repo)
	recorder := httptest.NewRecorder()
	req := notificationRequest(http.MethodPost, "/api/v1/notifications/"+item.ID+"/handle", `{"disposition":"closed"}`)
	req.SetPathValue("id", item.ID)

	handler.Handle(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}
