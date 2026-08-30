#!/usr/bin/env bash
# ==============================================================================
# Smart Clinic Queue Web App - Micro-Benchmark & High-Concurrency Stress Test Orchestrator
# Executes Go Domain Micro-Benchmarks (testing.B) and High-Concurrency Stress Runner
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

cd "${ROOT_DIR}"

export PORT="${PORT:-8081}"
export DATABASE_URL="${DATABASE_URL:-postgres://postgres:postgrespassword@localhost:5433/clinic_queue_test?sslmode=disable}"
export NATS_URL="${NATS_URL:-nats://localhost:4222}"
export JWT_SECRET="${JWT_SECRET:-super-secret-clinic-jwt-key-change-in-prod}"
export JWT_EXPIRATION_HOURS="${JWT_EXPIRATION_HOURS:-24}"
export CASBIN_MODEL_PATH="${CASBIN_MODEL_PATH:-config/rbac_model.conf}"
export CASBIN_POLICY_PATH="${CASBIN_POLICY_PATH:-config/rbac_policy.csv}"
export API_BASE_URL="http://localhost:${PORT}"

echo "========================================================================="
echo "  SMART CLINIC QUEUE - BENCHMARK & STRESS TEST ORCHESTRATOR"
echo "  Target API URL : ${API_BASE_URL} (Isolated Test Port)"
echo "  Database URL   : ${DATABASE_URL} (Isolated Test DB)"
echo "  NATS URL       : ${NATS_URL}"
echo "========================================================================="

# 1. Verify Docker Containers (Postgres 18 & NATS)
echo ""
echo "--> Checking Docker container infrastructure..."
if ! docker compose ps | grep -q "clinic-postgres"; then
    echo "Starting Docker Compose services..."
    docker compose up -d
fi

echo "--> Waiting for PostgreSQL 18 health check on port 5433..."
MAX_DB_RETRIES=20
DB_COUNT=0
until docker exec clinic-postgres pg_isready -U postgres -d clinic_queue >/dev/null 2>&1; do
    sleep 0.5
    DB_COUNT=$((DB_COUNT + 1))
    if [[ ${DB_COUNT} -ge ${MAX_DB_RETRIES} ]]; then
        echo "Error: PostgreSQL failed to become healthy within 10 seconds."
        exit 1
    fi
done
echo "--> PostgreSQL 18 is healthy and accepting connections."

# Auto-provision test database if using Docker container
if docker compose ps | grep -q "clinic-postgres"; then
    echo "--> Ensuring test database 'clinic_queue_test' exists in Docker container..."
    docker exec clinic-postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'clinic_queue_test'" | grep -q 1 || \
        docker exec clinic-postgres psql -U postgres -c "CREATE DATABASE clinic_queue_test;"
    echo "--> Test database 'clinic_queue_test' verified."
fi

SERVER_PID=""

cleanup() {
    if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
        echo ""
        echo "--> Stopping background API server (PID: ${SERVER_PID})..."
        kill -TERM "${SERVER_PID}" 2>/dev/null || true
        wait "${SERVER_PID}" 2>/dev/null || true
        echo "--> API Server stopped gracefully."
    fi
}
trap cleanup EXIT INT TERM

# 2. Check if server is already running on target port
if curl -s -f "${API_BASE_URL}/health" >/dev/null 2>&1; then
    echo "--> Detected API server already active on ${API_BASE_URL}."
else
    echo "--> Launching API server on port ${PORT}..."
    go run ./cmd/api/main.go &
    SERVER_PID=$!
    
    echo "--> Waiting for API server to become healthy..."
    MAX_RETRIES=30
    COUNT=0
    until curl -s -f "${API_BASE_URL}/health" >/dev/null 2>&1; do
        sleep 0.5
        COUNT=$((COUNT + 1))
        if [[ ${COUNT} -ge ${MAX_RETRIES} ]]; then
            echo "Error: Server failed to start within 15 seconds."
            exit 1
        fi
    done
    echo "--> API server is healthy and ready!"
fi

# 3. Execute Go Micro-Benchmarks
echo ""
echo "========================================================================="
echo "  PHASE 1: Executing Domain Algorithm Micro-Benchmarks (testing.B)"
echo "========================================================================="
go test -bench=. -benchmem -run=^$ ./internal/core/domain/...

# 4. Execute High-Concurrency Stress Test Runner
echo ""
echo "========================================================================="
echo "  PHASE 2: Executing High-Concurrency Stress & Contention Runner"
echo "========================================================================="
go run "${SCRIPT_DIR}/stress_benchmark_runner.go"

echo ""
echo "========================================================================="
echo "  Benchmark and Stress Test Orchestration Completed Successfully!"
echo "========================================================================="
