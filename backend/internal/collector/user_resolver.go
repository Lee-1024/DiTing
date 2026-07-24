package collector

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"diting/backend/internal/audit"
)

const UnsetAuditUID uint32 = 4294967295

type UserResolver interface {
	Username(uid uint32) string
}

type PasswdUserResolver struct {
	users map[uint32]string
}

// NewPasswdUserResolver 创建并初始化 New Passwd User Resolver 实例。
func NewPasswdUserResolver(path string) (*PasswdUserResolver, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	resolver := &PasswdUserResolver{users: map[uint32]string{}}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		uid, err := strconv.ParseUint(parts[2], 10, 32)
		if err != nil {
			continue
		}
		resolver.users[uint32(uid)] = parts[0]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	slog.Info("passwd resolver loaded", "path", path, "users", len(resolver.users))
	return resolver, nil
}

// Username 处理 Username 相关逻辑。
func (r *PasswdUserResolver) Username(uid uint32) string {
	if r == nil {
		return ""
	}
	return r.users[uid]
}

type IdentityWriter struct {
	resolver UserResolver
	next     EventWriter
}

// NewIdentityWriter 创建并初始化 New Identity Writer 实例。
func NewIdentityWriter(resolver UserResolver, next EventWriter) *IdentityWriter {
	return &IdentityWriter{resolver: resolver, next: next}
}

// Write 写入 Write 数据。
func (w *IdentityWriter) Write(ctx context.Context, events []audit.Event) error {
	enriched := make([]audit.Event, len(events))
	unresolvedUsername := 0
	unresolvedLoginUsername := 0
	for i, event := range events {
		enriched[i] = w.enrich(event)
		if enriched[i].Username == "" {
			unresolvedUsername++
		}
		if enriched[i].LoginUsername == "" && enriched[i].AUID != UnsetAuditUID {
			unresolvedLoginUsername++
		}
	}
	if unresolvedUsername > 0 || unresolvedLoginUsername > 0 {
		slog.Warn("collector user resolution incomplete", "events", len(events), "unresolved_username", unresolvedUsername, "unresolved_login_username", unresolvedLoginUsername)
	}
	return w.next.Write(ctx, enriched)
}

// enrich 处理 enrich 相关逻辑。
func (w *IdentityWriter) enrich(event audit.Event) audit.Event {
	if w.resolver == nil {
		return event
	}
	if event.Username == "" {
		event.Username = w.resolver.Username(event.UID)
	}
	if event.LoginUsername == "" && event.AUID != UnsetAuditUID {
		event.LoginUsername = w.resolver.Username(event.AUID)
	}
	return event
}
