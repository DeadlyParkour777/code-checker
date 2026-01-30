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

register_payload=$(printf '{"username":"%s","password":"%s"}' "$ADMIN_USERNAME" "$ADMIN_PASSWORD")

resp_file=$(mktemp)
status=$(curl -s -o "$resp_file" -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -d "$register_payload" \
  "$BASE_URL/auth/register" || true)

if [ "$status" = "201" ]; then
  echo "registered admin user: $ADMIN_USERNAME"
elif [ "$status" = "400" ] || [ "$status" = "500" ]; then
  echo "register returned status $status (user may already exist)"
else
  echo "unexpected register status $status"
fi

rm -f "$resp_file"

DB_CONTAINER=${DB_CONTAINER:-db}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-code_checker_db}

update_sql="UPDATE users SET role='admin' WHERE username='${ADMIN_USERNAME}';"
check_sql="SELECT COUNT(*) FROM users WHERE username='${ADMIN_USERNAME}' AND role='admin';"

exec_psql() {
  docker exec -e PGPASSWORD="$DB_PASSWORD" "$DB_CONTAINER" \
    psql -U "$DB_USER" -d "$DB_NAME" -t -c "$1"
}

exec_psql "$update_sql" >/dev/null
rows=$(exec_psql "$check_sql" | tr -d '[:space:]')

if [ "$rows" = "1" ]; then
  echo "admin role applied for: $ADMIN_USERNAME"
else
  echo "admin role update did not apply (user missing?)" >&2
  exit 1
fi
