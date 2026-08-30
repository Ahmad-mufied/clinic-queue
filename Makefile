.DEFAULT_GOAL := help

# ==============================================================================
# SMART CLINIC QUEUE - DEVELOPER MAKEFILE
# ==============================================================================

.PHONY: help setup infra-up infra-down infra-restart infra-logs \
        dev-api dev-web dev build-api build-web build \
        test test-cover test-html test-bench test-e2e test-stress vet clean

help: ## Tampilkan daftar command yang tersedia
	@echo "======================================================================"
	@echo "  Smart Clinic Queue & Analytics Platform - Makefile Commands"
	@echo "======================================================================"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo "======================================================================"

# ------------------------------------------------------------------------------
# 1. SETUP & INFRASTRUCTURE (Docker Compose: PostgreSQL 18 & NATS JetStream)
# ------------------------------------------------------------------------------

setup: ## Siapkan file .env dan install dependencies frontend
	@if [ ! -f .env ]; then cp .env.example .env && echo "Created .env from .env.example"; fi
	@cd web && npm install
	@echo "Setup complete."

infra-up: ## Jalankan container PostgreSQL 18 (:5433) & NATS (:4222) di background
	docker compose up -d

infra-down: ## Matikan container infrastruktur
	docker compose down

infra-restart: ## Restart container infrastruktur
	docker compose restart

infra-logs: ## Pantau real-time log dari container database & message broker
	docker compose logs -f

# ------------------------------------------------------------------------------
# 2. LOCAL DEVELOPMENT (Air Hot-Reload & Next.js)
# ------------------------------------------------------------------------------

dev-api: ## Jalankan backend Go dengan Air (Hot-Reload otomatis)
	@which air > /dev/null || (echo "Air tidak ditemukan. Menginstall air..." && go install github.com/air-verse/air@latest)
	air

dev-web: ## Jalankan frontend Next.js 15 dev server (:3000)
	cd web && npm run dev

dev: ## Petunjuk menjalankan backend dan frontend secara bersamaan
	@echo "\033[33mUntuk menjalankan fullstack development:\033[0m"
	@echo "  Terminal 1 (Infrastruktur): \033[36mmake infra-up\033[0m"
	@echo "  Terminal 2 (Backend Go)   : \033[36mmake dev-api\033[0m  (Hot reload via Air di :8080)"
	@echo "  Terminal 3 (Frontend)     : \033[36mmake dev-web\033[0m  (Next.js di :3000)"

# ------------------------------------------------------------------------------
# 3. BUILD & PACKAGING
# ------------------------------------------------------------------------------

build-api: ## Kompilasi standalone binary Go untuk production
	@mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/api ./cmd/api
	@echo "Backend binary built at bin/api"

build-web: ## Build frontend Next.js 15 untuk production
	cd web && npm run build

build: build-api build-web ## Build seluruh sistem (Backend & Frontend)

# ------------------------------------------------------------------------------
# 4. QUALITY ASSURANCE, TESTING & BENCHMARKS
# ------------------------------------------------------------------------------

test: ## Jalankan semua unit tests dengan race detector & coverage profile
	go test -v -race -coverprofile=coverage.out ./...

test-cover: ## Tampilkan rincian coverage fungsi di terminal
	@if [ ! -f coverage.out ]; then $(MAKE) test; fi
	go tool cover -func=coverage.out

test-html: ## Tampilkan visualisasi coverage interaktif di browser
	@if [ ! -f coverage.out ]; then $(MAKE) test; fi
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

test-bench: ## Jalankan micro-benchmark Greedy Scheduling Engine
	go test -bench=. -benchmem ./internal/core/domain/...

test-e2e: ## Jalankan master automated E2E regression suite (70 live scenarios)
	./scripts/testing/e2e/regression_e2e_test.sh

test-stress: ## Jalankan high-concurrency stress testing harness (500 joins, 50 calls)
	./scripts/testing/stress/run_stress_benchmark.sh

vet: ## Analisis statis kode Go dengan go vet
	go vet ./...

# ------------------------------------------------------------------------------
# 5. MAINTENANCE & CLEANUP
# ------------------------------------------------------------------------------

clean: ## Bersihkan temporary build binaries, logs, dan coverage output
	rm -rf tmp bin coverage.out coverage.html web/.next
	@echo "Cleaned build artifacts and temporary files."
