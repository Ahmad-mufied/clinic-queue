# Smart Clinic Queue: Automation & Operational Scripts

This directory contains automated testing, regression, benchmarking, and operational scripts for the Smart Clinic Queue application.

---

## 1. Directory Structure

```
scripts/
└── testing/                      # Dedicated automated testing & benchmark domain
    ├── README.md                 # Testing domain quick guide
    ├── e2e/                      # End-to-End master regression test suite
    │   ├── e2e_runner.go         # 70-scenario multi-persona live HTTP runner
    │   └── regression_e2e_test.sh# E2E test orchestrator shell script
    └── stress/                   # High-concurrency stress test harness
        ├── stress_benchmark_runner.go # High-load stress runner (500 joins, lock race, SSE)
        └── run_stress_benchmark.sh    # Stress test orchestrator shell script
```

---

## 2. Quick Execution Commands

### 2.1 Run Master E2E Regression Suite (70 Scenarios across PRD 01–05)
```bash
# From code/web-app root:
bash scripts/testing/e2e/regression_e2e_test.sh
```

### 2.2 Run High-Concurrency Stress Test Suite (5,000+ RPS)
```bash
# From code/web-app root:
bash scripts/testing/stress/run_stress_benchmark.sh
```

### 2.3 Run Go Algorithmic Micro-Benchmarks
```bash
# From code/web-app root:
go test -bench=. -benchmem ./internal/core/domain/...
```

---

## 3. Comprehensive Documentation & Maintenance Handbook

For complete architectural diagrams, operational SOPs, adjustment playbooks, and step-by-step guides on adding new test scenarios:

Reference Guide: [`docs/tech/TESTING-AND-BENCHMARK-GUIDE.md`](../docs/tech/TESTING-AND-BENCHMARK-GUIDE.md)

Formal verification reports:
- [`docs/reports/qa-regression-e2e-report.md`](../docs/reports/qa-regression-e2e-report.md) (E2E Regression Audit Report)
- [`docs/reports/benchmark-and-stress-report.md`](../docs/reports/benchmark-and-stress-report.md) (Performance & Stress Audit Report)
