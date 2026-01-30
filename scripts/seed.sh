#!/bin/sh
set -eu

if [ -f ./.env ]; then
  set -a
  . ./.env
  set +a
fi

BASE_URL=${BASE_URL:-http://localhost:${GATEWAY_PORT:-8000}}
ADMIN_USERNAME=${ADMIN_USERNAME:-admin}
ADMIN_PASSWORD=${ADMIN_PASSWORD:-admin123}

login_payload=$(printf '{"username":"%s","password":"%s"}' "$ADMIN_USERNAME" "$ADMIN_PASSWORD")

login_resp=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "$login_payload" \
  "$BASE_URL/auth/login")

token=$(printf '%s' "$login_resp" | sed -n 's/.*"access_token":"\([^"]*\)".*/\1/p')
if [ -z "$token" ]; then
  echo "failed to get access token; login response: $login_resp" >&2
  exit 1
fi

auth_header="Authorization: Bearer $token"

create_problem() {
  title=$1
  description=$2
  resp=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "$auth_header" \
    -d "{\"title\":\"$title\",\"description\":\"$description\"}" \
    "$BASE_URL/problems")
  id=$(printf '%s' "$resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
  if [ -z "$id" ]; then
    echo "failed to create problem; response: $resp" >&2
    exit 1
  fi
  echo "$id"
}

create_testcase() {
  problem_id=$1
  input_data=$2
  output_data=$3
  resp=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "$auth_header" \
    -d "{\"input_data\":\"$input_data\",\"output_data\":\"$output_data\"}" \
    "$BASE_URL/problems/$problem_id/testcases")
  id=$(printf '%s' "$resp" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
  if [ -z "$id" ]; then
    echo "failed to create test case; response: $resp" >&2
    exit 1
  fi
}

p1_id=$(create_problem "Sum A+B" "Return sum of two integers.")
create_testcase "$p1_id" "1 2" "3"
create_testcase "$p1_id" "10 5" "15"

p2_id=$(create_problem "Max of Two" "Return maximum of two integers.")
create_testcase "$p2_id" "1 2" "2"
create_testcase "$p2_id" "10 5" "10"

echo "seed completed"
