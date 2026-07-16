package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"diting/backend/internal/audit"
	"diting/backend/internal/auth"
	ch "diting/backend/internal/clickhouse"
	"diting/backend/internal/collector"
	"diting/backend/internal/config"
	"diting/backend/internal/hostasset"
	"diting/backend/internal/operationlog"
	"diting/backend/internal/postgres"
	"diting/backend/internal/riskstatus"
	"diting/backend/internal/rule"
	"diting/backend/internal/server"
	"diting/backend/internal/systemconfig"
	"diting/backend/internal/useradmin"
)

func main() {
	slog.SetDefault(slog.New(newLogHandler(os.Stdout)))
	mode, cfgPath := parseArgs(os.Args)
	slog.Info("process starting", "mode", mode, "config", cfgPath)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("load config failed", "config", cfgPath, "error", err)
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	slog.Info("config loaded",
		"mode", mode,
		"server_port", cfg.Server.Port,
		"postgres", fmt.Sprintf("%s:%d/%s", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.Database),
		"clickhouse", cfg.ClickHouse.Addr,
		"clickhouse_database", cfg.ClickHouse.Database,
	)

	if mode == "collector" || mode == "collector-once" {
		client := ch.NewHTTPClient(ch.HTTPConfig{
			URL:      ch.HTTPURLFromAddress(cfg.ClickHouse.Addr),
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
		})
		pool, err := postgres.Connect(context.Background(), cfg.Postgres)
		if err != nil {
			slog.Error("connect postgres failed", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database, "error", err)
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		slog.Info("postgres connected", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database)
		defer pool.Close()
		ruleRepository := rule.NewPostgresRepository(pool)
		systemConfigRepository := systemconfig.NewPostgresRepository(pool)
		writer := newRefreshingRuleWriter(client, repositoryRuleProvider{repository: ruleRepository})
		writer.SetCollectorFilterProvider(systemConfigRepository)
		if err := writer.Refresh(context.Background()); err != nil {
			slog.Error("load rules failed", "error", err)
			fmt.Fprintf(os.Stderr, "load rules: %v\n", err)
			os.Exit(1)
		}
		if mode == "collector" {
			go writer.RefreshLoop(context.Background(), 30*time.Second)
		}
		eventWriter := collector.EventWriter(writer)
		if cfg.Collector.PasswdFile != "" {
			resolver, err := collector.NewPasswdUserResolver(cfg.Collector.PasswdFile)
			if err != nil {
				slog.Error("load passwd file failed", "path", cfg.Collector.PasswdFile, "error", err)
				fmt.Fprintf(os.Stderr, "load passwd file: %v\n", err)
				os.Exit(1)
			}
			slog.Info("passwd file loaded", "path", cfg.Collector.PasswdFile)
			eventWriter = collector.NewIdentityWriter(resolver, writer)
		}
		hostMetadata := collector.ResolveHostMetadata(cfg.Collector.HostID, cfg.Collector.HostName)
		slog.Info("collector host metadata resolved", "host_id", hostMetadata.ID, "host_name", hostMetadata.Name)
		eventWriter = collector.NewHostMetadataWriter(hostMetadata, eventWriter)

		fileCollector := collector.NewFileCollector(cfg.Collector.TetragonLogFile, cfg.Collector.BatchSize, eventWriter)
		slog.Info("collector starting", "mode", mode, "tetragon_log_file", cfg.Collector.TetragonLogFile, "passwd_file", cfg.Collector.PasswdFile, "batch_size", cfg.Collector.BatchSize, "flush_interval_seconds", cfg.Collector.FlushIntervalSeconds)
		if mode == "collector-once" {
			err = fileCollector.RunOnce(context.Background())
		} else {
			err = fileCollector.Tail(context.Background(), time.Duration(cfg.Collector.FlushIntervalSeconds)*time.Second)
		}
		if err != nil {
			slog.Error("collector stopped with error", "error", err)
			fmt.Fprintf(os.Stderr, "run collector: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if mode == "migrate-clickhouse" {
		client := ch.NewHTTPClient(ch.HTTPConfig{
			URL:      ch.HTTPURLFromAddress(cfg.ClickHouse.Addr),
			Database: "",
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
		})
		files, err := migrationFiles(filepath.Join("migrations", "clickhouse"))
		if err != nil {
			slog.Error("list clickhouse migrations failed", "error", err)
			fmt.Fprintf(os.Stderr, "list clickhouse migrations: %v\n", err)
			os.Exit(1)
		}
		for _, sqlPath := range files {
			data, err := os.ReadFile(sqlPath)
			if err != nil {
				slog.Error("read clickhouse migration failed", "path", sqlPath, "error", err)
				fmt.Fprintf(os.Stderr, "read clickhouse migration %s: %v\n", sqlPath, err)
				os.Exit(1)
			}
			slog.Info("executing clickhouse migration", "path", sqlPath)
			if err := client.ExecuteStatements(context.Background(), string(data)); err != nil {
				slog.Error("execute clickhouse migration failed", "path", sqlPath, "error", err)
				fmt.Fprintf(os.Stderr, "execute clickhouse migration %s: %v\n", sqlPath, err)
				os.Exit(1)
			}
		}
		slog.Info("clickhouse migrations completed", "count", len(files))
		return
	}

	if mode == "migrate-postgres" {
		pool, err := postgres.Connect(context.Background(), cfg.Postgres)
		if err != nil {
			slog.Error("connect postgres failed", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database, "error", err)
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		slog.Info("postgres connected", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database)
		defer pool.Close()
		files, err := migrationFiles(filepath.Join("migrations", "postgres"))
		if err != nil {
			slog.Error("list postgres migrations failed", "error", err)
			fmt.Fprintf(os.Stderr, "list postgres migrations: %v\n", err)
			os.Exit(1)
		}
		for _, sqlPath := range files {
			slog.Info("executing postgres migration", "path", sqlPath)
			if err := postgres.ExecuteMigrationFile(context.Background(), pool, sqlPath); err != nil {
				slog.Error("execute postgres migration failed", "path", sqlPath, "error", err)
				fmt.Fprintf(os.Stderr, "execute postgres migration %s: %v\n", sqlPath, err)
				os.Exit(1)
			}
		}
		slog.Info("postgres migrations completed", "count", len(files))
		return
	}

	if mode == "clear-test-data" {
		client := ch.NewHTTPClient(ch.HTTPConfig{
			URL:      ch.HTTPURLFromAddress(cfg.ClickHouse.Addr),
			Database: "",
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
		})
		auditTable := "audit_events"
		if cfg.ClickHouse.Database != "" {
			auditTable = cfg.ClickHouse.Database + "." + auditTable
		}
		slog.Warn("clearing clickhouse audit events", "table", auditTable)
		if err := client.Execute(context.Background(), "TRUNCATE TABLE IF EXISTS "+auditTable); err != nil {
			slog.Error("clear clickhouse audit events failed", "table", auditTable, "error", err)
			fmt.Fprintf(os.Stderr, "clear clickhouse audit events: %v\n", err)
			os.Exit(1)
		}
		pool, err := postgres.Connect(context.Background(), cfg.Postgres)
		if err != nil {
			slog.Error("connect postgres failed", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database, "error", err)
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		defer pool.Close()
		slog.Warn("clearing postgres risk dispositions", "table", "diting_risk_dispositions")
		if _, err := pool.Exec(context.Background(), "DELETE FROM diting_risk_dispositions"); err != nil {
			slog.Error("clear postgres risk dispositions failed", "error", err)
			fmt.Fprintf(os.Stderr, "clear postgres risk dispositions: %v\n", err)
			os.Exit(1)
		}
		slog.Info("test data cleared")
		return
	}

	clickHouseClient := ch.NewHTTPClient(ch.HTTPConfig{
		URL:      ch.HTTPURLFromAddress(cfg.ClickHouse.Addr),
		Database: cfg.ClickHouse.Database,
		Username: cfg.ClickHouse.Username,
		Password: cfg.ClickHouse.Password,
	})
	auditRepository := ch.NewAuditRepository(clickHouseClient)
	pool, err := postgres.Connect(context.Background(), cfg.Postgres)
	if err != nil {
		slog.Error("connect postgres failed", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database, "error", err)
		fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
		os.Exit(1)
	}
	slog.Info("postgres connected", "host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database)
	defer pool.Close()
	ruleRepository := rule.NewPostgresRepository(pool)
	statsRepository := ch.NewStatsRepository(clickHouseClient, ruleRepository)
	userRepository := auth.NewPostgresUserRepository(pool)
	authService := auth.NewService(userRepository, auth.Config{Secret: cfg.JWT.Secret, ExpiresHours: cfg.JWT.ExpiresHours})
	operationRepository := operationlog.NewPostgresRepository(pool)
	hostAssetRepository := hostasset.NewPostgresRepository(pool)
	riskStatusRepository := riskstatus.NewPostgresRepository(pool)
	systemConfigRepository := systemconfig.NewPostgresRepository(pool)
	userAdminRepository := useradmin.NewPostgresRepository(pool)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("api server listening", "addr", addr)
	if err := http.ListenAndServe(addr, server.NewRouter(auditRepository, ruleRepository, statsRepository, authService, operationRepository, hostAssetRepository, riskStatusRepository, systemConfigRepository, userAdminRepository)); err != nil {
		slog.Error("api server stopped with error", "addr", addr, "error", err)
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
}

func newLogHandler(writer io.Writer) slog.Handler {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.String(slog.TimeKey, attr.Value.Time().In(location).Format("2006-01-02 15:04:05.000 MST"))
			}
			return attr
		},
	})
}

func migrationFiles(dir string) ([]string, error) {
	pattern := filepath.Join(dir, "*.sql")
	return filepath.Glob(pattern)
}

func parseArgs(args []string) (string, string) {
	mode := "api"
	configPath := "./configs/config.example.yaml"
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "api", "collector", "collector-once", "migrate-clickhouse", "migrate-postgres", "clear-test-data":
			mode = args[i]
		case "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		default:
			configPath = args[i]
		}
	}
	return mode, configPath
}

type eventSink interface {
	WriteEvents(ctx context.Context, events []audit.Event) error
}

type ruleProvider interface {
	Rules(ctx context.Context) ([]rule.Rule, error)
}

type collectorFilterProvider interface {
	GetCollectorFilter(ctx context.Context) (systemconfig.CollectorFilterConfig, error)
}

type repositoryRuleProvider struct {
	repository rule.Repository
}

func (p repositoryRuleProvider) Rules(ctx context.Context) ([]rule.Rule, error) {
	return p.repository.List(ctx)
}

type refreshingRuleWriter struct {
	sink           eventSink
	provider       ruleProvider
	filterProvider collectorFilterProvider
	mu             sync.RWMutex
	rules          []rule.Rule
	filter         collectorNoiseFilter
}

func newRefreshingRuleWriter(sink eventSink, provider ruleProvider) *refreshingRuleWriter {
	return &refreshingRuleWriter{sink: sink, provider: provider, rules: []rule.Rule{}}
}

type collectorNoiseFilter struct {
	Enabled               bool
	IgnoreProcessNames    []string
	IgnoreCommandKeywords []string
	IgnoreUsers           []string
	KeepSeverities        []string
}

func (w *refreshingRuleWriter) SetNoiseFilter(filter collectorNoiseFilter) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.filter = filter
}

func (w *refreshingRuleWriter) SetCollectorFilterProvider(provider collectorFilterProvider) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.filterProvider = provider
}

func collectorNoiseFilterFromSystemConfig(cfg systemconfig.CollectorFilterConfig) collectorNoiseFilter {
	return collectorNoiseFilter{
		Enabled:               cfg.Enabled,
		IgnoreProcessNames:    cfg.IgnoreProcessNames,
		IgnoreCommandKeywords: cfg.IgnoreCommandKeywords,
		IgnoreUsers:           cfg.IgnoreUsers,
		KeepSeverities:        cfg.KeepSeverities,
	}
}

func (w *refreshingRuleWriter) Refresh(ctx context.Context) error {
	rules, err := w.provider.Rules(ctx)
	if err != nil {
		slog.Error("refresh rules failed", "error", err)
		return err
	}
	var filter collectorNoiseFilter
	if w.filterProvider != nil {
		config, err := w.filterProvider.GetCollectorFilter(ctx)
		if err != nil {
			slog.Error("refresh collector filter failed", "error", err)
			return err
		}
		filter = collectorNoiseFilterFromSystemConfig(config)
	}
	w.mu.Lock()
	w.rules = rules
	if w.filterProvider != nil {
		w.filter = filter
	}
	w.mu.Unlock()
	slog.Info("rules refreshed", "count", len(rules))
	return nil
}

func (w *refreshingRuleWriter) RefreshLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Refresh(ctx); err != nil {
				slog.Error("refresh rules loop failed", "error", err)
			}
		}
	}
}

func (w *refreshingRuleWriter) Write(ctx context.Context, events []audit.Event) error {
	w.mu.RLock()
	rules := make([]rule.Rule, len(w.rules))
	copy(rules, w.rules)
	filter := w.filter
	w.mu.RUnlock()

	enriched := make([]audit.Event, 0, len(events))
	for _, event := range events {
		next := rule.ApplyRules(event, rules)
		if filter.ShouldDrop(next) {
			continue
		}
		enriched = append(enriched, next)
	}
	if len(enriched) == 0 {
		slog.Info("events filtered", "events", len(events), "rules", len(rules))
		return nil
	}
	if err := w.sink.WriteEvents(ctx, enriched); err != nil {
		slog.Error("write events failed", "events", len(enriched), "rules", len(rules), "error", err)
		return err
	}
	slog.Info("events written", "events", len(enriched), "rules", len(rules))
	return nil
}

func (f collectorNoiseFilter) ShouldDrop(event audit.Event) bool {
	if !f.Enabled || f.shouldKeep(event) {
		return false
	}
	if containsFold(f.IgnoreProcessNames, event.ProcessName) {
		return true
	}
	if containsFold(f.IgnoreUsers, event.Username) || containsFold(f.IgnoreUsers, event.LoginUsername) {
		return true
	}
	cmdline := strings.ToLower(event.Cmdline)
	for _, keyword := range f.IgnoreCommandKeywords {
		if keyword != "" && strings.Contains(cmdline, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (f collectorNoiseFilter) shouldKeep(event audit.Event) bool {
	keepSeverities := f.KeepSeverities
	if len(keepSeverities) == 0 {
		keepSeverities = []string{"high", "critical"}
	}
	return containsFold(keepSeverities, event.Severity)
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}
