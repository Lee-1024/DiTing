#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
BIN_DIR="$ROOT_DIR/bin"
CONFIG="$ROOT_DIR/backend/configs/config.yaml"
ONLY=""

usage() {
  cat <<EOF
Usage: scripts/migrate-linux.sh [options]

Options:
  --config PATH        Backend config file. Default: backend/configs/config.yaml
  --only NAME          Run only one migration target: postgres or clickhouse
  -h, --help          Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)
      CONFIG="$2"
      shift 2
      ;;
    --only)
      ONLY="$2"
      shift 2
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

if [[ "$ONLY" != "" && "$ONLY" != "postgres" && "$ONLY" != "clickhouse" ]]; then
  echo "--only must be postgres or clickhouse" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Missing required command: go" >&2
  exit 1
fi

if [[ ! -f "$CONFIG" ]]; then
  echo "Config file not found: $CONFIG" >&2
  exit 1
fi

mkdir -p "$BIN_DIR" "$ROOT_DIR/.cache/go-build" "$ROOT_DIR/.cache/goenv"

export GOCACHE="$ROOT_DIR/.cache/go-build"
export GOTELEMETRY=off
export GOENV="$ROOT_DIR/.cache/goenv"

echo "Building backend binary..."
(cd "$BACKEND_DIR" && go build -o "$BIN_DIR/audit-server" ./cmd/audit-server)

if [[ "$ONLY" == "" || "$ONLY" == "postgres" ]]; then
  echo "Running PostgreSQL migrations..."
  (cd "$BACKEND_DIR" && "$BIN_DIR/audit-server" migrate-postgres --config "$CONFIG")
fi

if [[ "$ONLY" == "" || "$ONLY" == "clickhouse" ]]; then
  echo "Running ClickHouse migrations..."
  (cd "$BACKEND_DIR" && "$BIN_DIR/audit-server" migrate-clickhouse --config "$CONFIG")
fi

echo "Migrations completed."
