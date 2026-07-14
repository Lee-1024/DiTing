#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
BIN_DIR="$ROOT_DIR/bin"
CONFIG="$ROOT_DIR/backend/configs/config.yaml"
YES="0"

usage() {
  cat <<EOF
Usage: scripts/clear-test-data-linux.sh [options]

Options:
  --config PATH        Backend config file. Default: backend/configs/config.yaml
  --yes               Confirm destructive cleanup
  -h, --help          Show this help

This clears collected audit test data:
  - ClickHouse diting.audit_events
  - PostgreSQL diting_risk_dispositions
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)
      CONFIG="$2"
      shift 2
      ;;
    --yes)
      YES="1"
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

if [[ "$YES" != "1" ]]; then
  echo "This will delete collected audit test data from ClickHouse and risk dispositions from PostgreSQL." >&2
  echo "Re-run with --yes to continue." >&2
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

echo "Clearing collected test data..."
(cd "$BACKEND_DIR" && "$BIN_DIR/audit-server" clear-test-data --config "$CONFIG")

echo "Test data cleared."
