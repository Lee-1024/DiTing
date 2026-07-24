package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

type HTTPConfig struct {
	URL      string
	Database string
	Username string
	Password string
}

type HTTPClient struct {
	config HTTPConfig
	client *http.Client
}

// NewHTTPClient 创建并初始化 New HTTPClient 实例。
func NewHTTPClient(config HTTPConfig) *HTTPClient {
	return &HTTPClient{
		config: config,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// HTTPURLFromAddress 处理 HTTPURLFrom Address 相关逻辑。
func HTTPURLFromAddress(addr string) string {
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return addr
	}
	host, _, ok := strings.Cut(addr, ":")
	if !ok {
		return "http://" + addr + ":8123"
	}
	return "http://" + host + ":8123"
}

// WriteEvents 写入 Write Events 数据。
func (c *HTTPClient) WriteEvents(ctx context.Context, events []audit.Event) error {
	if len(events) == 0 {
		return nil
	}

	var body bytes.Buffer
	table := "audit_events"
	if c.config.Database != "" {
		table = c.config.Database + "." + table
	}
	body.WriteString("INSERT INTO " + table + " FORMAT JSONEachRow\n")
	for _, event := range events {
		if err := json.NewEncoder(&body).Encode(toClickHouseRow(event)); err != nil {
			return err
		}
	}

	if err := c.do(ctx, body.String()); err != nil {
		slog.Error("clickhouse write events failed", "url", c.config.URL, "database", c.config.Database, "events", len(events), "error", err)
		return err
	}
	slog.Info("clickhouse write events completed", "url", c.config.URL, "database", c.config.Database, "events", len(events))
	return nil
}

// Execute 处理 Execute 相关逻辑。
func (c *HTTPClient) Execute(ctx context.Context, query string) error {
	return c.do(ctx, query)
}

// ExecuteStatements 处理 Execute Statements 相关逻辑。
func (c *HTTPClient) ExecuteStatements(ctx context.Context, sql string) error {
	for _, statement := range splitStatements(sql) {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if err := c.Execute(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

// splitStatements 处理 split Statements 相关逻辑。
func splitStatements(sql string) []string {
	parts := strings.Split(sql, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			statements = append(statements, trimmed)
		}
	}
	return statements
}

// do 处理 do 相关逻辑。
func (c *HTTPClient) do(ctx context.Context, query string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.URL, strings.NewReader(query))
	if err != nil {
		return err
	}
	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clickhouse http status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// Query 处理 Query 相关逻辑。
func (c *HTTPClient) Query(ctx context.Context, query string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.URL, strings.NewReader(query))
	if err != nil {
		return nil, err
	}
	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("clickhouse http status %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}

type clickHouseRow struct {
	EventID           string   `json:"event_id"`
	EventTime         string   `json:"event_time"`
	EventDate         string   `json:"event_date"`
	IngestTime        string   `json:"ingest_time"`
	EventType         string   `json:"event_type"`
	Action            string   `json:"action"`
	Severity          string   `json:"severity"`
	RiskScore         uint8    `json:"risk_score"`
	Tags              []string `json:"tags"`
	HostName          string   `json:"host_name"`
	HostID            string   `json:"host_id"`
	HostIP            string   `json:"host_ip"`
	NodeName          string   `json:"node_name"`
	Namespace         string   `json:"namespace"`
	PodName           string   `json:"pod_name"`
	ContainerID       string   `json:"container_id"`
	ContainerName     string   `json:"container_name"`
	Image             string   `json:"image"`
	PID               uint32   `json:"pid"`
	PPID              uint32   `json:"ppid"`
	ProcessName       string   `json:"process_name"`
	BinaryPath        string   `json:"binary_path"`
	Cmdline           string   `json:"cmdline"`
	CWD               string   `json:"cwd"`
	ParentProcessName string   `json:"parent_process_name"`
	ParentBinaryPath  string   `json:"parent_binary_path"`
	ParentCmdline     string   `json:"parent_cmdline"`
	UID               uint32   `json:"uid"`
	GID               uint32   `json:"gid"`
	Username          string   `json:"username"`
	AUID              uint32   `json:"auid"`
	EUID              uint32   `json:"euid"`
	EGID              uint32   `json:"egid"`
	LoginUsername     string   `json:"login_username"`
	FilePath          string   `json:"file_path"`
	FileOperation     string   `json:"file_operation"`
	SrcIP             string   `json:"src_ip"`
	SrcPort           uint16   `json:"src_port"`
	DstIP             string   `json:"dst_ip"`
	DstPort           uint16   `json:"dst_port"`
	Protocol          string   `json:"protocol"`
	Domain            string   `json:"domain"`
	RuleIDs           []string `json:"rule_ids"`
	RuleNames         []string `json:"rule_names"`
	RuleMatches       string   `json:"rule_matches"`
	RawEvent          string   `json:"raw_event"`
}

// toClickHouseRow 处理 to Click House Row 相关逻辑。
func toClickHouseRow(event audit.Event) clickHouseRow {
	eventDate := event.EventDate
	if eventDate.IsZero() {
		eventDate = event.EventTime
	}
	tags := nonNilStrings(event.Tags)
	ruleIDs := nonNilStrings(event.RuleIDs)
	ruleNames := nonNilStrings(event.RuleNames)
	ruleMatches, _ := json.Marshal(event.RuleMatches)
	return clickHouseRow{
		EventID: event.EventID, EventTime: formatDateTime64(event.EventTime), EventDate: eventDate.Format("2006-01-02"), IngestTime: formatDateTime64(event.IngestTime),
		EventType: event.EventType, Action: event.Action, Severity: event.Severity, RiskScore: event.RiskScore, Tags: tags,
		HostName: event.HostName, HostID: event.HostID, HostIP: event.HostIP, NodeName: event.NodeName,
		Namespace: event.Namespace, PodName: event.PodName, ContainerID: event.ContainerID, ContainerName: event.ContainerName, Image: event.Image,
		PID: event.PID, PPID: event.PPID, ProcessName: event.ProcessName, BinaryPath: event.BinaryPath, Cmdline: event.Cmdline, CWD: event.CWD,
		ParentProcessName: event.ParentProcessName, ParentBinaryPath: event.ParentBinaryPath, ParentCmdline: event.ParentCmdline,
		UID: event.UID, GID: event.GID, Username: event.Username, AUID: event.AUID, EUID: event.EUID, EGID: event.EGID, LoginUsername: event.LoginUsername,
		FilePath: event.FilePath, FileOperation: event.FileOperation,
		SrcIP: event.SrcIP, SrcPort: event.SrcPort, DstIP: event.DstIP, DstPort: event.DstPort, Protocol: event.Protocol, Domain: event.Domain,
		RuleIDs: ruleIDs, RuleNames: ruleNames, RuleMatches: string(ruleMatches), RawEvent: event.RawEvent,
	}
}

// formatDateTime64 格式化 format Date Time64 以便展示或写入。
func formatDateTime64(value time.Time) string {
	if value.IsZero() {
		value = time.Unix(0, 0).UTC()
	}
	return value.UTC().Format("2006-01-02 15:04:05.000")
}

// nonNilStrings 处理 non Nil Strings 相关逻辑。
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
