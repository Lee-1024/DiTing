package notification

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"diting/backend/internal/collectorhealth"
)

type HealthRepository interface {
	List(context.Context, time.Time) ([]collectorhealth.Heartbeat, error)
}

func ReconcileHealth(ctx context.Context, notifications Repository, health HealthRepository, now time.Time) error {
	items, err := health.List(ctx, now)
	if err != nil {
		return err
	}
	for _, item := range items {
		hostID := strings.TrimSpace(item.HostID)
		if hostID == "" {
			hostID = strings.TrimSpace(item.HostName)
		}
		if hostID == "" {
			continue
		}
		displayName := strings.TrimSpace(item.HostName)
		if displayName == "" {
			displayName = hostID
		}
		offlineKey := "collector:offline:" + hostID
		tetragonKey := "collector:tetragon:" + hostID
		warningKey := "collector:warning:" + hostID
		if item.Status == "offline" {
			if _, err := notifications.Upsert(ctx, Input{
				Type: TypeCollector, DedupeKey: offlineKey, SourceID: hostID,
				Title: "采集节点离线：" + displayName, Description: "Collector 心跳超时",
				Severity: "critical", Target: "/settings/collector-health",
			}); err != nil {
				return err
			}
			if err := resolveNotifications(ctx, notifications, tetragonKey, warningKey); err != nil {
				return err
			}
			continue
		}
		if err := notifications.Resolve(ctx, offlineKey); err != nil {
			return err
		}
		message := strings.TrimSpace(item.LastError)
		if message == "" {
			message = strings.TrimSpace(item.Message)
		}
		if strings.Contains(strings.ToLower(message), "tetragon") {
			if _, err := notifications.Upsert(ctx, Input{
				Type: TypeTetragon, DedupeKey: tetragonKey, SourceID: hostID,
				Title: "Tetragon 服务异常：" + displayName, Description: message,
				Severity: collectorNotificationSeverity(item.HealthLevel), Target: "/settings/collector-health",
			}); err != nil {
				return err
			}
			if err := notifications.Resolve(ctx, warningKey); err != nil {
				return err
			}
			continue
		}
		if item.HealthLevel == "warning" || item.HealthLevel == "critical" {
			title := "采集服务预警：" + displayName
			if item.HealthLevel == "critical" {
				title = "采集服务异常：" + displayName
			}
			if _, err := notifications.Upsert(ctx, Input{
				Type: TypeCollector, DedupeKey: warningKey, SourceID: hostID,
				Title: title, Description: message,
				Severity: collectorNotificationSeverity(item.HealthLevel), Target: "/settings/collector-health",
			}); err != nil {
				return err
			}
			if err := notifications.Resolve(ctx, tetragonKey); err != nil {
				return err
			}
			continue
		}
		if err := resolveNotifications(ctx, notifications, tetragonKey, warningKey); err != nil {
			return err
		}
	}
	return nil
}

func collectorNotificationSeverity(healthLevel string) string {
	if healthLevel == "critical" {
		return "critical"
	}
	return "warning"
}

func resolveNotifications(ctx context.Context, notifications Repository, keys ...string) error {
	for _, key := range keys {
		if err := notifications.Resolve(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func RunHealthReconciler(ctx context.Context, notifications Repository, health HealthRepository, interval time.Duration) {
	if notifications == nil || health == nil {
		return
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	reconcile := func() {
		if err := ReconcileHealth(ctx, notifications, health, time.Now().UTC()); err != nil {
			slog.Error("reconcile notification health failed", "error", err)
		}
	}
	reconcile()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			reconcile()
		}
	}
}
