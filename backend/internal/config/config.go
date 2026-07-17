package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server     ServerConfig
	JWT        JWTConfig
	Postgres   PostgresConfig
	ClickHouse ClickHouseConfig
	Collector  CollectorConfig
}

type ServerConfig struct {
	Port int
	Mode string
}

type JWTConfig struct {
	Secret       string
	ExpiresHours int
}

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string
}

type ClickHouseConfig struct {
	Addr     string
	Database string
	Username string
	Password string
}

type CollectorConfig struct {
	InputMode                string
	TetragonLogFile          string
	TetragonGRPCAddr         string
	PasswdFile               string
	HostID                   string
	HostName                 string
	FlushIntervalSeconds     int
	BatchSize                int
	ReconnectIntervalSeconds int
}

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	cfg := Config{}
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			section = strings.TrimSuffix(line, ":")
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Config{}, fmt.Errorf("invalid config line %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if err := assignValue(&cfg, section, key, value); err != nil {
			return Config{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func assignValue(cfg *Config, section, key, value string) error {
	switch section {
	case "server":
		switch key {
		case "port":
			cfg.Server.Port = mustInt(key, value)
		case "mode":
			cfg.Server.Mode = value
		}
	case "jwt":
		switch key {
		case "secret":
			cfg.JWT.Secret = value
		case "expires_hours":
			cfg.JWT.ExpiresHours = mustInt(key, value)
		}
	case "postgres":
		switch key {
		case "host":
			cfg.Postgres.Host = value
		case "port":
			cfg.Postgres.Port = mustInt(key, value)
		case "database":
			cfg.Postgres.Database = value
		case "username":
			cfg.Postgres.Username = value
		case "password":
			cfg.Postgres.Password = value
		case "ssl_mode":
			cfg.Postgres.SSLMode = value
		}
	case "clickhouse":
		switch key {
		case "addr":
			cfg.ClickHouse.Addr = value
		case "database":
			cfg.ClickHouse.Database = value
		case "username":
			cfg.ClickHouse.Username = value
		case "password":
			cfg.ClickHouse.Password = value
		}
	case "collector":
		switch key {
		case "input_mode":
			cfg.Collector.InputMode = value
		case "tetragon_log_file":
			cfg.Collector.TetragonLogFile = value
		case "tetragon_grpc_addr":
			cfg.Collector.TetragonGRPCAddr = value
		case "passwd_file":
			cfg.Collector.PasswdFile = value
		case "host_id":
			cfg.Collector.HostID = value
		case "host_name":
			cfg.Collector.HostName = value
		case "flush_interval_seconds":
			cfg.Collector.FlushIntervalSeconds = mustInt(key, value)
		case "batch_size":
			cfg.Collector.BatchSize = mustInt(key, value)
		case "reconnect_interval_seconds":
			cfg.Collector.ReconnectIntervalSeconds = mustInt(key, value)
		}
	}
	return nil
}

func mustInt(key, value string) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("invalid integer for %s: %q", key, value))
	}
	return parsed
}
