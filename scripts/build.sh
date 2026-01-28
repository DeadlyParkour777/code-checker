#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/modules.sh"

for m in "${SERVICE_MODULES[@]}"; do
  echo "==> $m"
  ( cd "$ROOT_DIR/$m" && go build ./... )
done
