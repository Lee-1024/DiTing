#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/run"

stop_pid_file() {
  local name="$1"
  local file="$2"

  if [[ ! -f "$file" ]]; then
    return
  fi

  local pid
  pid="$(cat "$file")"
  if [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
    echo "Stopping $name PID $pid..."
    kill "$pid" >/dev/null 2>&1 || true
    for _ in {1..20}; do
      if ! kill -0 "$pid" >/dev/null 2>&1; then
        break
      fi
      sleep 0.2
    done
    if kill -0 "$pid" >/dev/null 2>&1; then
      echo "Force stopping $name PID $pid..."
      kill -9 "$pid" >/dev/null 2>&1 || true
    fi
  fi
  rm -f "$file"
}

stop_pid_file "DiTing Web" "$RUN_DIR/web.pid"
stop_pid_file "DiTing Collector" "$RUN_DIR/collector.pid"
stop_pid_file "DiTing API" "$RUN_DIR/api.pid"

echo "DiTing stopped."
