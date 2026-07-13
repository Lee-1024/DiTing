package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
)

func main() {
	mode, cfgPath := parseArgs(os.Args)

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	if mode == "collector" || mode == "collector-once" {
		client := ch.NewHTTPClient(ch.HTTPConfig{
			URL:      ch.HTTPURLFromAddress(cfg.ClickHouse.Addr),
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
		})
		pool, err := postgres.Connect(context.Background(), cfg.Postgres)
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		defer pool.Close()
		ruleRepository := rule.NewPostgresRepository(pool)
		writer := newRefreshingRuleWriter(client, repositoryRuleProvider{repository: ruleRepository})
		if err := writer.Refresh(context.Background()); err != nil {
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
				fmt.Fprintf(os.Stderr, "load passwd file: %v\n", err)
				os.Exit(1)
			}
			eventWriter = collector.NewIdentityWriter(resolver, writer)
		}

		fileCollector := collector.NewFileCollector(cfg.Collector.TetragonLogFile, cfg.Collector.BatchSize, eventWriter)
		if mode == "collector-once" {
			err = fileCollector.RunOnce(context.Background())
		} else {
			err = fileCollector.Tail(context.Background(), time.Duration(cfg.Collector.FlushIntervalSeconds)*time.Second)
		}
		if err != nil {
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
			fmt.Fprintf(os.Stderr, "list clickhouse migrations: %v\n", err)
			os.Exit(1)
		}
		for _, sqlPath := range files {
			data, err := os.ReadFile(sqlPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "read clickhouse migration %s: %v\n", sqlPath, err)
				os.Exit(1)
			}
			if err := client.ExecuteStatements(context.Background(), string(data)); err != nil {
				fmt.Fprintf(os.Stderr, "execute clickhouse migration %s: %v\n", sqlPath, err)
				os.Exit(1)
			}
		}
		return
	}

	if mode == "migrate-postgres" {
		pool, err := postgres.Connect(context.Background(), cfg.Postgres)
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
			os.Exit(1)
		}
		defer pool.Close()
		files, err := migrationFiles(filepath.Join("migrations", "postgres"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "list postgres migrations: %v\n", err)
			os.Exit(1)
		}
		for _, sqlPath := range files {
			if err := postgres.ExecuteMigrationFile(context.Background(), pool, sqlPath); err != nil {
				fmt.Fprintf(os.Stderr, "execute postgres migration %s: %v\n", sqlPath, err)
				os.Exit(1)
			}
		}
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
		fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	ruleRepository := rule.NewPostgresRepository(pool)
	statsRepository := ch.NewStatsRepository(clickHouseClient, ruleRepository)
	userRepository := auth.NewPostgresUserRepository(pool)
	authService := auth.NewService(userRepository, auth.Config{Secret: cfg.JWT.Secret, ExpiresHours: cfg.JWT.ExpiresHours})
	operationRepository := operationlog.NewPostgresRepository(pool)
	hostAssetRepository := hostasset.NewPostgresRepository(pool)
	riskStatusRepository := riskstatus.NewPostgresRepository(pool)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	if err := http.ListenAndServe(addr, server.NewRouter(auditRepository, ruleRepository, statsRepository, authService, operationRepository, hostAssetRepository, riskStatusRepository)); err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
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
		case "api", "collector", "collector-once", "migrate-clickhouse", "migrate-postgres":
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

type repositoryRuleProvider struct {
	repository rule.Repository
}

func (p repositoryRuleProvider) Rules(ctx context.Context) ([]rule.Rule, error) {
	return p.repository.List(ctx)
}

type refreshingRuleWriter struct {
	sink     eventSink
	provider ruleProvider
	mu       sync.RWMutex
	rules    []rule.Rule
}

func newRefreshingRuleWriter(sink eventSink, provider ruleProvider) *refreshingRuleWriter {
	return &refreshingRuleWriter{sink: sink, provider: provider, rules: []rule.Rule{}}
}

func (w *refreshingRuleWriter) Refresh(ctx context.Context) error {
	rules, err := w.provider.Rules(ctx)
	if err != nil {
		return err
	}
	w.mu.Lock()
	w.rules = rules
	w.mu.Unlock()
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
			_ = w.Refresh(ctx)
		}
	}
}

func (w *refreshingRuleWriter) Write(ctx context.Context, events []audit.Event) error {
	w.mu.RLock()
	rules := make([]rule.Rule, len(w.rules))
	copy(rules, w.rules)
	w.mu.RUnlock()

	enriched := make([]audit.Event, len(events))
	for i, event := range events {
		enriched[i] = rule.ApplyRules(event, rules)
	}
	return w.sink.WriteEvents(ctx, enriched)
}
