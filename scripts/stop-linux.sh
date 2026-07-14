#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RUN_DIR="$ROOT_DIR/run"
FRONTEND_DIR="$ROOT_DIR/frontend"
WEB_PORT="5174"

usage() {
  cat <<EOF
Usage: scripts/stop-linux.sh [options]

Options:
  --web-port PORT     Frontend port used for targeted Vite cleanup. Default: 5174
  -h, --help          Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --web-port)
      WEB_PORT="$2"
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

stop_pid_file() {
  local name="$1"
  local file="$2"
  local pgid_file="${file%.pid}.pgid"

  if [[ ! -f "$file" && ! -f "$pgid_file" ]]; then
    return
  fi

  local pid
  pid=""
  if [[ -f "$file" ]]; then
    pid="$(cat "$file")"
  fi

  local pgid
  pgid=""
  if [[ -f "$pgid_file" ]]; then
    pgid="$(cat "$pgid_file")"
  elif [[ -n "$pid" ]]; then
    pgid="$pid"
  fi

  if [[ -n "$pgid" ]] && kill -0 "-$pgid" >/dev/null 2>&1; then
    echo "Stopping $name process group $pgid..."
    kill "-$pgid" >/dev/null 2>&1 || true
    for _ in {1..20}; do
      if ! kill -0 "-$pgid" >/dev/null 2>&1; then
        break
      fi
      sleep 0.2
    done
    if kill -0 "-$pgid" >/dev/null 2>&1; then
      echo "Force stopping $name process group $pgid..."
      kill -9 "-$pgid" >/dev/null 2>&1 || true
    fi
  elif [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
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
  rm -f "$file" "$pgid_file"
}

stop_pid_file "DiTing Web" "$RUN_DIR/web.pid"
stop_pid_file "DiTing Collector" "$RUN_DIR/collector.pid"
stop_pid_file "DiTing API" "$RUN_DIR/api.pid"

web_port_pids() {
  if command -v lsof >/dev/null 2>&1; then
    lsof -tiTCP:"$WEB_PORT" -sTCP:LISTEN 2>/dev/null || true
    return
  fi
  if command -v ss >/dev/null 2>&1; then
    ss -ltnp "( sport = :$WEB_PORT )" 2>/dev/null | sed -nE 's/.*pid=([0-9]+).*/\1/p' | sort -u
  fi
}

stop_web_leftovers() {
  local pids
  pids="$(web_port_pids)"
  if [[ -z "$pids" ]]; then
    return
  fi
  while read -r pid; do
    [[ -z "$pid" ]] && continue
    local args
    args="$(ps -p "$pid" -o args= 2>/dev/null || true)"
    if [[ "$args" == *"$FRONTEND_DIR"* || ( "$args" == *"vite"* && "$args" == *"$WEB_PORT"* ) ]]; then
      echo "Stopping leftover DiTing Web listener PID $pid on port $WEB_PORT..."
      kill "$pid" >/dev/null 2>&1 || true
      sleep 0.5
      if kill -0 "$pid" >/dev/null 2>&1; then
        echo "Force stopping leftover DiTing Web listener PID $pid..."
        kill -9 "$pid" >/dev/null 2>&1 || true
      fi
    else
      echo "Port $WEB_PORT is still used by another process; leaving it running: PID $pid $args" >&2
    fi
  done <<< "$pids"
}

stop_web_leftovers

echo "DiTing stopped."
