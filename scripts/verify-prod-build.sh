#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
COMPOSE_FILE="${ROOT_DIR}/docker-compose.prod.yml"
PROJECT_NAME="${PROJECT_NAME:-augr-prod}"
AUTH_TOKEN="${AUTH_TOKEN:?AUTH_TOKEN must be set}"

compose() {
    docker compose --project-name "$PROJECT_NAME" -f "$COMPOSE_FILE" "$@"
}

wait_for_postgres() {
    echo "Waiting for postgres..."
    for i in $(seq 1 30); do
        if compose exec -T postgres pg_isready -U augr >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    echo "Postgres did not become ready" >&2
    exit 1
}

wait_for_app_health() {
    echo "Waiting for app health..."
    for i in $(seq 1 60); do
        response=$(wget -qO- http://127.0.0.1:8080/healthz 2>/dev/null || true)
        if python3 -c "
import json, sys
body = json.loads(sys.argv[1])
sys.exit(0 if body.get(\"status\") == \"ok\" and body.get(\"db\") == \"ok\" and body.get(\"redis\") == \"ok\" else 1)
" "$response" 2>/dev/null; then
            return 0
        fi
        sleep 1
    done
    echo "App did not become healthy" >&2
    exit 1
}

echo "=== Building production images ==="
compose build

echo "=== Starting services ==="
compose up -d

echo "=== Waiting for Postgres ==="
wait_for_postgres

echo "=== Running migrations ==="
migration_files=()
while IFS= read -r f; do
    migration_files+=("$f")
done < <(find "${ROOT_DIR}/migrations" -maxdepth 1 -type f -name '*.up.sql' -printf '%f\n' | sort)

for migration in "${migration_files[@]}"; do
    echo "Applying migration: ${migration}"
    compose exec -T postgres psql -U augr -d augr -f "/migrations/${migration}"
done

echo "=== Verifying schema version ==="
SCHEMA_VERSION=$(compose exec -T postgres psql -U augr -d augr -t -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1" | tr -d '[:space:]')
EXPECTED_VERSION="${migration_files[-1]%%%%_*}"
if [ "$SCHEMA_VERSION" != "$EXPECTED_VERSION" ]; then
    echo "schema version mismatch after migrations: got ${SCHEMA_VERSION}, expected ${EXPECTED_VERSION}" >&2
    exit 1
fi

echo "=== Waiting for app health ==="
wait_for_app_health

echo "=== Smoke-testing API ==="
curl -sf -H "Authorization: Bearer ${AUTH_TOKEN}" http://127.0.0.1:8080/api/v1/strategies | python3 -c "
import json, sys
data = json.load(sys.stdin)
print(f'Strategies: {len(data)}')
"

echo "=== Production build verification passed ==="
