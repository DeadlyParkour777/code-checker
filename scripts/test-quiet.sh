#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/modules.sh"

for m in "${MODULES[@]}"; do
  echo "==> $m"
  ( cd "$ROOT_DIR/$m" && go test -v ./... ) | awk ' 
    /^(=== RUN|--- PASS|--- FAIL|PASS$|FAIL$|\?\s|ok\s|FAIL\t|panic:)/ {print; next} 
    /^\t.*\.go:[0-9]+:/ {print; next} 
  '
done
