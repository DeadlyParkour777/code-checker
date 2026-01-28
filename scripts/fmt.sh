#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
rg --files -g "*.go" "$ROOT_DIR" | xargs gofmt -w
