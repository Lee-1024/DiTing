#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
LOG_DIR="$ROOT_DIR/logs"
RUN_DIR="$ROOT_DIR/run"
BIN_DIR="$ROOT_DIR/bin"

CONFIG="$ROOT_DIR/backend/configs/config.yaml"
WEB_PORT="5174"
API_PORT="8089"
SKIP_FRONTEND="0"
SKIP_COLLECTOR="0"
RUN_MIGRATIONS="0"

usage() {
  cat <<EOF
Usage: scripts/start-linux.sh [options]

Options:
  --config PATH        Backend config file. Default: backend/configs/config.yaml
  --web-port PORT     Frontend port. Default: 5174
  --api-port PORT     API port used for port checks. Default: 8089
  --skip-frontend     Start API and Collector only
  --skip-collector    Start API and frontend only
  --migrate           Run PostgreSQL and ClickHouse migrations before start
  -h, --help          Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)
      CONFIG="$2"
      shift 2
      ;;
    --web-port)
      WEB_PORT="$2"
      shift 2
      ;;
    --api-port)
      API_PORT="$2"
      shift 2
      ;;
    --skip-frontend)
      SKIP_FRONTEND="1"
      shift
      ;;
    --skip-collector)
      SKIP_COLLECTOR="1"
      shift
      ;;
    --migrate)
      RUN_MIGRATIONS="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

case "$CONFIG" in
  /*) ;;
  *) CONFIG="$ROOT_DIR/$CONFIG" ;;
esac

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required command: $1" >&2
    exit 1
  fi
}

port_in_use() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltn "( sport = :$port )" | tail -n +2 | grep -q .
    return $?
  fi
  netstat -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]$port$"
}

start_process() {
  local name="$1"
  local pid_file="$2"
  local log_name="$3"
  shift 3

  local stdout="$LOG_DIR/$log_name.out.log"
  local stderr="$LOG_DIR/$log_name.err.log"

  echo "Starting $name..."
  nohup "$@" >"$stdout" 2>"$stderr" &
  local pid=$!
  echo "$pid" >"$pid_file"
  sleep 0.5
  if ! kill -0 "$pid" >/dev/null 2>&1; then
    echo "$name failed to start. Check logs:" >&2
    echo "  $stdout" >&2
    echo "  $stderr" >&2
    exit 1
  fi
  echo "  PID $pid"
}

require_command go
if [[ "$SKIP_FRONTEND" != "1" ]]; then
  require_command npm
fi

if [[ ! -f "$CONFIG" ]]; then
  echo "Config file not found: $CONFIG" >&2
  exit 1
fi

if port_in_use "$API_PORT"; then
  echo "API port $API_PORT is already in use. Run scripts/stop-linux.sh or stop the existing process first." >&2
  exit 1
fi

if [[ "$SKIP_FRONTEND" != "1" ]] && port_in_use "$WEB_PORT"; then
  echo "Web port $WEB_PORT is already in use. Use --web-port or stop the existing process first." >&2
  exit 1
fi

mkdir -p "$LOG_DIR" "$RUN_DIR" "$BIN_DIR" "$ROOT_DIR/.cache/go-build" "$ROOT_DIR/.cache/goenv"

export GOCACHE="$ROOT_DIR/.cache/go-build"
export GOTELEMETRY=off
export GOENV="$ROOT_DIR/.cache/goenv"

echo "Building backend binary..."
(cd "$BACKEND_DIR" && go build -o "$BIN_DIR/audit-server" ./cmd/audit-server)

if [[ "$RUN_MIGRATIONS" == "1" ]]; then
  echo "Running migrations..."
  (cd "$BACKEND_DIR" && "$BIN_DIR/audit-server" migrate-clickhouse --config "$CONFIG")
  (cd "$BACKEND_DIR" && "$BIN_DIR/audit-server" migrate-postgres --config "$CONFIG")
fi

if [[ "$SKIP_FRONTEND" != "1" ]]; then
  if [[ ! -d "$FRONTEND_DIR/node_modules" ]]; then
    echo "Installing frontend dependencies..."
    (cd "$FRONTEND_DIR" && npm install)
  fi
fi

start_process "DiTing API" "$RUN_DIR/api.pid" "api" \
  "$BIN_DIR/audit-server" api --config "$CONFIG"

if [[ "$SKIP_COLLECTOR" != "1" ]]; then
  start_process "DiTing Collector" "$RUN_DIR/collector.pid" "collector" \
    "$BIN_DIR/audit-server" collector --config "$CONFIG"
fi

if [[ "$SKIP_FRONTEND" != "1" ]]; then
  start_process "DiTing Web" "$RUN_DIR/web.pid" "web" \
    npm --prefix "$FRONTEND_DIR" run dev -- --port "$WEB_PORT" --strictPort
fi

cat <<EOF

DiTing started.
API:        http://127.0.0.1:$API_PORT/healthz
Web:        http://127.0.0.1:$WEB_PORT
Config:     $CONFIG
Logs:       $LOG_DIR
PID files:  $RUN_DIR/*.pid

Stop with:
  scripts/stop-linux.sh
EOF
