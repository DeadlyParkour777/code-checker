#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/modules.sh"

for m in "${MODULES[@]}"; do
  echo "==> $m"
  ( cd "$ROOT_DIR/$m" && go test -v ./... )
done
