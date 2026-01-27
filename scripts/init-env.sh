#!/usr/bin/env bash
set -euo pipefail

copy_if_missing() {
  local src="$1"
  local dst="$2"
  if [ ! -f "$src" ]; then
    echo "missing template: $src" >&2
    exit 1
  fi
  if [ ! -f "$dst" ]; then
    cp "$src" "$dst"
    echo "created $dst"
  else
    echo "exists  $dst"
  fi
}

copy_if_missing .env.example .env
copy_if_missing services/auth_service/.env.example services/auth_service/.env
copy_if_missing services/problem_service/.env.example services/problem_service/.env
copy_if_missing services/submission_service/.env.example services/submission_service/.env
copy_if_missing services/judge_service/.env.example services/judge_service/.env
copy_if_missing services/result_service/.env.example services/result_service/.env
copy_if_missing services/gateway/.env.example services/gateway/.env
