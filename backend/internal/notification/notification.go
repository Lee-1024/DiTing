package notification

import (
	"context"
	"errors"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

const (
	TypeEnforcement = "enforcement"
	TypeCollector   = "collector"
	TypeTetragon    = "tetragon"
	StatusOpen      = "open"
	StatusResolved  = "resolved"
)

type Notification struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	DedupeKey   string     `json:"-"`
	SourceID    string     `json:"sourceId,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Severity    string     `json:"severity"`
	Target      string     `json:"target"`
	Status      string     `json:"status"`
	Disposition string     `json:"disposition,omitempty"`
	HandledBy   string     `json:"handledBy,omitempty"`
	HandledAt   *time.Time `json:"handledAt,omitempty"`
	Read        bool       `json:"read"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
}

type Input struct {
	Type        string
	DedupeKey   string
	SourceID    string
	Title       string
	Description string
	Severity    string
	Target      string
}

type Counts struct {
	Unread  int `json:"unread"`
	Pending int `json:"pending"`
	All     int `json:"all"`
}

type ListResult struct {
	Items  []Notification `json:"items"`
	Counts Counts         `json:"counts"`
}

type Repository interface {
	Upsert(context.Context, Input) (Notification, error)
	Resolve(context.Context, string) error
	List(context.Context, string, string, int) (ListResult, error)
	MarkRead(context.Context, string, string) error
	MarkAllRead(context.Context, string) error
	Handle(context.Context, string, string, string) error
}

var ErrNotFound = errors.New("notification not found")
var ErrInvalidDisposition = errors.New("invalid notification disposition")

func NormalizeView(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "unread":
		return "unread"
	case "pending":
		return "pending"
	case "all":
		return "all"
	default:
		return "unread"
	}
}

func NormalizeDisposition(value string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "confirmed", "false_positive", "ignored":
		return strings.TrimSpace(strings.ToLower(value)), nil
	default:
		return "", ErrInvalidDisposition
	}
}

func IsEnforcementEvent(event audit.Event) bool {
	for _, tag := range event.Tags {
		if tag == "diting-enforcement" {
			return true
		}
	}
	return false
}

func EnforcementInput(event audit.Event) Input {
	user := strings.TrimSpace(event.LoginUsername)
	if user == "" {
		user = strings.TrimSpace(event.Username)
	}
	if user == "" {
		user = "未知用户"
	}
	command := strings.TrimSpace(event.Cmdline)
	if command == "" {
		command = strings.TrimSpace(event.ProcessName)
	}
	if command == "" {
		command = strings.TrimSpace(event.Action)
	}
	if command == "" {
		command = "未知命令"
	}
	description := user + " 执行 " + command + " 已被拦截"
	if target := strings.TrimSpace(event.FilePath); target != "" {
		description += "，目标 " + target
	}
	return Input{
		Type: TypeEnforcement, DedupeKey: "enforcement:" + event.EventID, SourceID: event.EventID,
		Title: "拦截策略触发", Description: description, Severity: event.Severity,
		Target: "/audit/events?eventId=" + event.EventID,
	}
}
