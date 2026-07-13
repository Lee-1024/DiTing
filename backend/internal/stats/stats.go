package stats

import (
	"context"
	"time"
)

type Overview struct {
	TotalEvents    uint64 `json:"totalEvents"`
	HighRiskEvents uint64 `json:"highRiskEvents"`
	ActiveHosts    uint64 `json:"activeHosts"`
	ActiveRules    uint64 `json:"activeRules"`
}

type TrendPoint struct {
	Time  string `json:"time"`
	Count uint64 `json:"count"`
}

type TopItem struct {
	Name  string `json:"name"`
	Count uint64 `json:"count"`
}

type CommandItem struct {
	ProcessName   string `json:"processName"`
	Cmdline       string `json:"cmdline"`
	Username      string `json:"username"`
	LoginUsername string `json:"loginUsername"`
	Count         uint64 `json:"count"`
	FirstSeen     string `json:"firstSeen"`
	LastSeen      string `json:"lastSeen"`
}

type UserAuditItem struct {
	Username       string `json:"username"`
	CommandCount   uint64 `json:"commandCount"`
	ActiveHosts    uint64 `json:"activeHosts"`
	HighRiskEvents uint64 `json:"highRiskEvents"`
	FirstSeen      string `json:"firstSeen"`
	LastSeen       string `json:"lastSeen"`
}

type HostAuditItem struct {
	HostName       string `json:"hostName"`
	CommandCount   uint64 `json:"commandCount"`
	ActiveUsers    uint64 `json:"activeUsers"`
	HighRiskEvents uint64 `json:"highRiskEvents"`
	FirstSeen      string `json:"firstSeen"`
	LastSeen       string `json:"lastSeen"`
}

type Query struct {
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Keyword   string
	Username  string
}

type Repository interface {
	Overview(ctx context.Context, query Query) (Overview, error)
	EventTrend(ctx context.Context, query Query) ([]TrendPoint, error)
	TopCommands(ctx context.Context, query Query) ([]TopItem, error)
	CommandStats(ctx context.Context, query Query) ([]CommandItem, error)
	UserAudits(ctx context.Context, query Query) ([]UserAuditItem, error)
	HostAudits(ctx context.Context, query Query) ([]HostAuditItem, error)
}
