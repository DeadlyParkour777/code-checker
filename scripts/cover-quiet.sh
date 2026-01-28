#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/modules.sh"

for m in "${MODULES[@]}"; do
  echo "==> $m"
  ( cd "$ROOT_DIR/$m" && go test -cover ./... ) | awk ' 
    /^\?\s/ {next} 
    /coverage: 0\.0%/ {next} 
    {print} 
  '
done
