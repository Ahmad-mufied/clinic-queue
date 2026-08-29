# Implementation Plan: 01 - Authentication & Casbin RBAC Foundation
**File:** `docs/plan/01-auth-rbac-plan.md`  
**Status:** ✅ **100% COMPLETED & OFFICIALLY SIGNED OFF**  
**Target Feature:** [`docs/prd/01-authentication-rbac.md`](../prd/01-authentication-rbac.md) & [`docs/tech/01-auth-rbac-tech.md`](../tech/01-auth-rbac-tech.md)  
**Architecture:** Hexagonal Architecture (Ports & Adapters) in Go 1.27 + PostgreSQL 18  
**Completion Date:** 2026-08-29  

---

## 1. Objective & Scope

Establish the core project infrastructure and implement the **Authentication (JWT) and Authorization (Casbin RBAC)** foundation. This serves as the secure, decoupled backbone for all subsequent features (Patient Queue, Doctor Workspace, and Admin Analytics).

All Core UseCases and Inbound Handlers are decoupled using **Inbound & Outbound Ports (Go Interfaces)**, achieving **100% Statement Test Coverage** with the **Table-Driven Test Pattern**.

---

## 2. Technical Deliverables & Completion Checklist

- [x] **Phase 1.1: Project Setup & Infrastructure**
  - [x] `go.mod` (Go 1.27 runtime with Echo v4, pgx/v5, Casbin v2, Goose v3, golang-jwt v5, bcrypt).
  - [x] `config/config.go` & `config/config_test.go` (Centralized env parser, 100% statement coverage).
  - [x] `.env` and `.env.example` configurations.
  - [x] `docker-compose.yml` (PostgreSQL 18 on port 5433 with `/var/lib/postgresql` volume + NATS JetStream on port 4222).

- [x] **Phase 1.2: Database Schema & Migrations**
  - [x] `migrations/00001_initial_schema.sql` (Tables: `users`, `doctors`, `queue_tickets`, `consultation_sessions`, `audit_logs`).
  - [x] `migrations/00002_seed_demo_data.sql` (Pre-seeded Doctor A [3m], Doctor B [4m], patient_john, patient_lucas, admin with bcrypt `password123`).
  - [x] `internal/adapters/outbound/postgres/migration.go` (Embedded Goose auto-migrator executed on boot).

- [x] **Phase 1.3: Hexagonal Core Layer (Domain, Ports & UseCases)**
  - [x] `internal/core/domain/user.go` (`User` entity, `Role` enum with `IsValid()`, `JWTCustomClaims`).
  - [x] `internal/core/domain/errors.go` (Domain sentinel errors: `ErrInvalidCredentials`, `ErrUsernameTaken`, etc.).
  - [x] `internal/core/ports/inbound/auth_port.go` (`AuthUseCase` Inbound Port).
  - [x] `internal/core/ports/outbound/user_repo_port.go` (`UserRepositoryPort` Outbound SPI Port).
  - [x] `internal/core/usecase/auth_usecase.go` (Pure business logic: bcrypt hash verification, HMAC-SHA256 JWT generation).
  - [x] `internal/core/usecase/auth_usecase_test.go` (100% table-driven unit tests with closure-based mock ports).

- [x] **Phase 1.4: Inbound Adapters (Echo HTTP & Casbin RBAC Middleware)**
  - [x] `config/rbac_model.conf` (RESTful RBAC model with `keyMatch2` & `regexMatch`).
  - [x] `config/rbac_policy.csv` (Casbin route policies with role inheritance `g, role, public`).
  - [x] `internal/adapters/inbound/middleware/jwt_auth.go` (Bearer token extractor & context claims helper).
  - [x] `internal/adapters/inbound/middleware/casbin_rbac.go` (Casbin authorization enforcer).
  - [x] `internal/adapters/inbound/middleware/casbin_rbac_test.go` (100% table-driven middleware tests).
  - [x] `internal/adapters/inbound/http/auth_handler.go` (`/login`, `/register`, `/me`).
  - [x] `internal/adapters/inbound/http/auth_handler_test.go` (100% table-driven HTTP handler tests).

- [x] **Phase 1.5: Outbound Adapters (PostgreSQL 18 via pgx/v5)**
  - [x] `internal/adapters/outbound/postgres/db.go` (`pgxpool.Pool` initialization).
  - [x] `internal/adapters/outbound/postgres/user_repo.go` (PostgreSQL implementation of `UserRepositoryPort`).

- [x] **Phase 1.6: Server Bootstrap / Composition Root**
  - [x] `cmd/api/main.go` (Wires dependencies, runs migrations, initializes Casbin enforcer, registers middleware, handles graceful shutdown).

---

## 3. Comprehensive Quality & Testing Report

### 3.1 Unit Test Statement Coverage Report (`go test -v -race`)

```text
ok      clinic-queue/config                                 coverage: 100.0% of statements
ok      clinic-queue/internal/core/domain                   coverage: 100.0% of statements
ok      clinic-queue/internal/core/usecase                  coverage: 100.0% of statements
ok      clinic-queue/internal/adapters/inbound/http         coverage: 100.0% of statements
ok      clinic-queue/internal/adapters/inbound/middleware   coverage: 100.0% of statements
```

- **Core UseCase Coverage:** **100.0%** (All success paths, missing fields, user not found, bcrypt mismatch, database error propagation, username collisions, ID boundaries).
- **HTTP Handlers Coverage:** **100.0%** (200 OK, 201 Created, 400 Bad Request, 401 Unauthorized, 404 Not Found, 409 Conflict, 500 Internal Error).
- **Security Middlewares Coverage:** **100.0%** (Missing headers, invalid tokens, wrong signatures, expired tokens, role privilege matrix).
- **Race Condition Detection:** Enabled and passed with **`-race` flag (0 race conditions)**.

---

### 3.2 Live Manual Integration E2E Test Report (against PostgreSQL 18)

The backend server was booted alongside active PostgreSQL 18 and NATS containers. All 9 end-to-end integration scenarios were executed and verified via HTTP `curl`:

| Test ID | Scenario | HTTP Method & Path | Auth / Payload | HTTP Code | Verification Status |
| :---: | :--- | :--- | :--- | :---: | :---: |
| **E2E-01** | Server Healthcheck | `GET /health` | *None* | `200 OK` | ✅ **PASS** (`healthy`) |
| **E2E-02** | Invalid Credentials | `POST /api/auth/login` | `{"username":"doctor_a", "password":"bad"}` | `401 Unauthorized` | ✅ **PASS** (Rejection verified) |
| **E2E-03** | Doctor A Login | `POST /api/auth/login` | `{"username":"doctor_a", "password":"password123"}` | `200 OK` | ✅ **PASS** (JWT + `doctor_id: 1`) |
| **E2E-04** | Patient John Login | `POST /api/auth/login` | `{"username":"patient_john", "password":"password123"}` | `200 OK` | ✅ **PASS** (JWT + `role: patient`) |
| **E2E-05** | Admin Login | `POST /api/auth/login` | `{"username":"admin", "password":"password123"}` | `200 OK` | ✅ **PASS** (JWT + `role: admin`) |
| **E2E-06** | Patient Self-Registration | `POST /api/auth/register` | `{"username":"patient_alice_...", "name":"Alice"}` | `201 Created` | ✅ **PASS** (Stored in DB + JWT) |
| **E2E-07** | Profile Endpoint | `GET /api/auth/me` | `Authorization: Bearer <Doctor_Token>` | `200 OK` | ✅ **PASS** (Returned Dr. Sarah Adams) |
| **E2E-08** | Casbin RBAC Denial | `GET /api/admin/stats` | `Authorization: Bearer <Patient_Token>` | `403 Forbidden` | ✅ **PASS** (Access Denied) |
| **E2E-09** | Missing Token Denial | `GET /api/auth/me` | *None* | `401 Unauthorized` | ✅ **PASS** (Token required) |

---

### 3.3 Real-World Discovered Items & Resolutions

1. **Host Port Collision:** Mapped PostgreSQL container host port to `5433:5432` and updated `.env` to prevent collision with pre-existing host containers.
2. **PostgreSQL 18 Volume Standard:** Adjusted volume mount path to `/var/lib/postgresql` conforming with `pg_ctlcluster` layout in `postgres:18-alpine`.
3. **Casbin Role Inheritance:** Configured `g, doctor, public`, `g, patient, public`, `g, admin, public` in `config/rbac_policy.csv` enabling authenticated users to access general profile and public routes seamlessly.

---

## 4. Sign-Off & Readiness for Next Feature

| Milestone | Target | Actual Result | Sign-Off Status |
| :--- | :---: | :---: | :---: |
| **Hexagonal Layer Decoupling** | 100% | Pure Core (`domain`, `usecase`, `ports`) | ✅ **APPROVED** |
| **Code Statement Coverage** | 100% | **100.0%** across all target packages | ✅ **APPROVED** |
| **Live Integration Verification** | 100% | 9 / 9 Scenarios Passed | ✅ **APPROVED** |
| **Database Persistence & Migrations** | PostgreSQL 18 | Goose embedded migrations applied | ✅ **APPROVED** |

**Conclusion:** Feature 01 (Authentication & Casbin RBAC Foundation) is **officially completed, tested, and locked**. The system is 100% ready to proceed to **Feature 02: Patient Queue & Live Wait Countdown (Queue Engine & NATS JetStream SSE)**.

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Lead Architect | **Initial Baseline** | Initial implementation plan for Feature 01. |
| **v1.1.0** | 2026-08-29 | Lead Architect | **Testing Upgrade** | Added interface decoupling and comprehensive Table-Driven Test suites targeting 100% coverage for Service and Handler layers. |
| **v1.2.0** | 2026-08-29 | Backend Lead | **Execution Completed** | Implemented Hexagonal Core (Ports & UseCases), Inbound Adapters (Echo Handlers & Casbin RBAC), Outbound Adapters (PostgreSQL pgx), and achieved 100% Table-Driven unit test coverage. |
| **v1.3.0** | 2026-08-29 | Backend Lead | **Final Sign-Off Report** | Added comprehensive execution report, 100% unit test coverage breakdown, 9/9 E2E live test results, and official milestone sign-off. |
