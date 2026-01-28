#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

SERVICE_MODULES=(
  services/auth_service
  services/problem_service
  services/submission_service
  services/judge_service
  services/result_service
  services/gateway
)

OTHER_MODULES=(
  pkg
  migrations
)

MODULES=("${SERVICE_MODULES[@]}" "${OTHER_MODULES[@]}")
