#!/usr/bin/env bash
# Run from repo root. Ensures GOPROXY is usable if the shell had a bad value.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export GOPROXY="${GOPROXY:-https://proxy.golang.org,direct}"
export GOSUMDB="${GOSUMDB:-sum.golang.org}"

for m in user-service inventory-service order-service api-gateway; do
  echo ""
  echo "=== go test $m ==="
  (cd "$ROOT/$m" && go test ./...)
done

echo ""
echo "All modules passed."
