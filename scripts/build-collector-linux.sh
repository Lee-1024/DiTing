#!/usr/bin/env bash
set -euo pipefail

ARCH="amd64"
OUTPUT=""
RUN_TESTS="false"

usage() {
  cat <<'EOF'
Usage: scripts/build-collector-linux.sh [options]

Options:
  --arch amd64|arm64     Target Linux architecture. Default: amd64
  --output PATH          Output binary path. Default: dist/collector-linux-<arch>
  --test                 Run backend tests before building
  -h, --help             Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch)
      ARCH="${2:-}"
      shift 2
      ;;
    --output)
      OUTPUT="${2:-}"
      shift 2
      ;;
    --test)
      RUN_TESTS="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$ARCH" in
  amd64|arm64) ;;
  *)
    echo "Unsupported arch: $ARCH. Use amd64 or arm64." >&2
    exit 1
    ;;
esac

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND="$ROOT/backend"

if [[ -z "$OUTPUT" ]]; then
  OUTPUT="$ROOT/dist/collector-linux-$ARCH"
elif [[ "$OUTPUT" != /* ]]; then
  OUTPUT="$ROOT/$OUTPUT"
fi

mkdir -p "$(dirname "$OUTPUT")" "$ROOT/.cache/go-build" "$ROOT/.cache/goenv"

export GOCACHE="$ROOT/.cache/go-build"
export GOTELEMETRY="off"
export GOENV="$ROOT/.cache/goenv"
export CGO_ENABLED=0
export GOOS=linux
export GOARCH="$ARCH"

cd "$BACKEND"

if [[ "$RUN_TESTS" == "true" ]]; then
  echo "Running backend tests..."
  go test ./...
fi

echo "Building static Linux collector..."
echo "GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED"
go build -trimpath -ldflags="-s -w -extldflags '-static'" -o "$OUTPUT" ./cmd/audit-server

chmod +x "$OUTPUT"

echo
echo "Collector build complete."
echo "Binary: $OUTPUT"
echo
echo "Run on Linux:"
echo "  ./${OUTPUT##*/} collector --config ./config.yaml"
