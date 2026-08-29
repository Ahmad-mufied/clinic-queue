# Implementation Plan: 03 - Doctor Workspace & Shift Management
**File:** `docs/plan/03-doctor-workspace-plan.md`  
**Status:** ✅ **100% COMPLETED & OFFICIALLY SIGNED OFF**  
**Target Feature:** [`docs/prd/03-doctor-workspace.md`](../prd/03-doctor-workspace.md) & [`docs/tech/03-doctor-workspace-tech.md`](../tech/03-doctor-workspace-tech.md)  
**Architecture:** Hexagonal Architecture (Ports & Adapters) in Go 1.27 + NATS JetStream + PostgreSQL 18  
**Completion Date:** 2026-08-29  

---

## 1. Objective & Scope

Implement **Feature 03: Doctor Workspace & Consultation Workflow**:
1. **Shift Management:** Doctors can toggle `is_online` status (`POST /api/doctors/status`), updating their availability and broadcasting real-time SSE updates to patients.
2. **Atomic Patient Admission:** Concurrency-safe "Call Next Patient" (`POST /api/doctors/call-next`) using PostgreSQL row-level locks (`FOR UPDATE SKIP LOCKED`) inside database transactions, creating active `consultation_sessions` and marking tickets `IN_CONSULTATION`.
3. **Consultation Completion:** Doctors finish active consultations (`POST /api/doctors/finish`), marking sessions completed, tickets `COMPLETED`, resetting doctor room to `AVAILABLE`, and broadcasting queue updates via NATS JetStream & SSE.
4. **Doctor Workspace Query:** Doctors can query their active session and status (`GET /api/doctors/workspace`).
5. **100% Statement Coverage:** Comprehensive table-driven unit tests for UseCases, Domain, and HTTP Handlers.

---

## 2. Technical Deliverables & Completion Checklist

- [x] **Phase 3.1: Domain Entities & Errors**
  - [x] `internal/core/domain/doctor.go` (`ConsultationSession`, `ConsultationTicket`, `DoctorWorkspace`, `DoctorShiftResponse`, `ConsultationFinishResponse`).
  - [x] `internal/core/domain/errors.go` (Domain sentinel errors: `ErrDoctorOffline`, `ErrActiveConsultationExists`, `ErrNoActiveConsultation`, `ErrQueueEmpty`, `ErrDoctorNotFound`).
  - [x] `internal/core/domain/doctor_test.go` (100% statement coverage unit tests).

- [x] **Phase 3.2: Inbound & Outbound Ports**
  - [x] `internal/core/ports/inbound/doctor_port.go` (`DoctorUseCase` interface & DTOs).
  - [x] `internal/core/ports/outbound/doctor_repo_port.go` (Extended with `UpdateOnlineStatus`, `GetDoctorByID`, `GetActiveSessionByDoctorID`).
  - [x] `internal/core/ports/outbound/consultation_repo_port.go` (`ConsultationRepositoryPort` for atomic patient admission and session completion).

- [x] **Phase 3.3: Core UseCase Implementation**
  - [x] `internal/core/usecase/doctor_usecase.go` (`ToggleStatus`, `CallNextPatient`, `FinishConsultation`, `GetWorkspace`, with NATS JetStream event publishing).
  - [x] `internal/core/usecase/doctor_usecase_test.go` (100% Statement Coverage table-driven tests with closure-based mock ports).

- [x] **Phase 3.4: Outbound Adapters (PostgreSQL 18 & NATS JetStream)**
  - [x] `internal/adapters/outbound/postgres/consultation_repo.go` (Atomic `SELECT ... FOR UPDATE SKIP LOCKED` transaction for `CallNextPatient`, session closure for `FinishConsultation`).
  - [x] `internal/adapters/outbound/postgres/doctor_repo.go` (Implemented `UpdateOnlineStatus` and `GetActiveSessionByDoctorID`).

- [x] **Phase 3.5: Inbound Adapters (Echo HTTP Handlers & RBAC)**
  - [x] `internal/adapters/inbound/http/doctor_handler.go` (`POST /api/doctors/status`, `POST /api/doctors/call-next`, `POST /api/doctors/finish`, `GET /api/doctors/workspace`).
  - [x] `internal/adapters/inbound/http/doctor_handler_test.go` (100% Statement Coverage table-driven HTTP tests).
  - [x] `config/rbac_policy.csv` (Enforced doctor route access under Casbin RBAC).

- [x] **Phase 3.6: Server Bootstrap / Composition Root**
  - [x] `cmd/api/main.go` (Wired ConsultationRepo, DoctorUseCase, DoctorHandler, and registered routes).

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

- **Doctor Domain Logic:** **100.0%** statement coverage.
- **Doctor UseCase:** **100.0%** statement coverage (shift toggles, call-next, finish, workspace, error branches).
- **Doctor HTTP Handler:** **100.0%** statement coverage (all status codes 200, 400, 401, 403, 404, 409, 500).
- **Race Condition Detection:** **0 race conditions** with `-race` flag enabled.

---

### 3.2 Live Manual Integration & Real-Time SSE Test Report

The live server was booted alongside active PostgreSQL 18 and NATS JetStream containers. Full end-to-end integration scenarios were executed and verified:

| Test ID | Scenario | HTTP Method & Path | Payload / Conditions | Result | Verification Status |
| :---: | :--- | :--- | :--- | :---: | :---: |
| **E2E-DOC-01** | Doctor Authentication | `POST /api/auth/login` | `doctor_a`, `doctor_b` | `200 OK` | ✅ **PASS** (Tokens issued) |
| **E2E-DOC-02** | Doctor Workspace Initial | `GET /api/doctors/workspace` | `Bearer <DocA_Token>` | `200 OK` | ✅ **PASS** (Status: `AVAILABLE`) |
| **E2E-DOC-03** | Call Next Patient (Atomic) | `POST /api/doctors/call-next` | Doctor A free, queue has A-01 | `200 OK` | ✅ **PASS** (Ticket A-01 admitted) |
| **E2E-DOC-04** | Double Call Guard | `POST /api/doctors/call-next` | Doctor A already in consultation | `409 Conflict` | ✅ **PASS** (`Active consultation already in progress`) |
| **E2E-DOC-05** | Doctor B Call Next | `POST /api/doctors/call-next` | Doctor B free, queue has A-02 | `200 OK` | ✅ **PASS** (Ticket A-02 admitted) |
| **E2E-DOC-06** | Public Queue Status | `GET /api/queue/status` | Both doctors busy | `200 OK` | ✅ **PASS** (Both `IN_CONSULTATION`) |
| **E2E-DOC-07** | Finish Consultation | `POST /api/doctors/finish` | Doctor A finishes John Doe | `200 OK` | ✅ **PASS** (Reset to `AVAILABLE`) |
| **E2E-DOC-08** | Call Next Empty Queue | `POST /api/doctors/call-next` | 0 patients waiting | `200 OK` | ✅ **PASS** (`Queue is empty`) |
| **E2E-DOC-09** | Toggle Shift to Offline | `POST /api/doctors/status` | `{"is_online": false}` | `200 OK` | ✅ **PASS** (Status: `OFFLINE`) |
| **E2E-DOC-10** | Call While Offline Guard | `POST /api/doctors/call-next` | Doctor is offline | `400 Bad Request` | ✅ **PASS** (`Doctor must be online`) |
| **E2E-DOC-11** | NATS & SSE Broadcasting | `GET /api/events` | Real-time event stream | Streamed | ✅ **PASS** (`TICKET_CALLED`, `TICKET_FINISHED`, `DOCTOR_STATUS_CHANGED`) |

---

## 4. Sign-Off & Readiness for Feature 04

| Milestone | Target | Actual Result | Sign-Off Status |
| :--- | :---: | :---: | :---: |
| **Concurrency Row-Level Locking** | `FOR UPDATE SKIP LOCKED` | Verified atomic pop & 0 race conditions | ✅ **APPROVED** |
| **Code Statement Coverage** | 100% | **100.0%** across core usecases and handlers | ✅ **APPROVED** |
| **Real-Time NATS/SSE Streaming** | Live Stream | Real-time SSE delivery for doctor transitions | ✅ **APPROVED** |
| **Database Persistence & State** | PostgreSQL 18 | `consultation_sessions` & `queue_tickets` sync | ✅ **APPROVED** |

**Conclusion:** Feature 03 (Doctor Workspace & Shift Management) is **officially completed, tested, and locked**. Ready to proceed to **Feature 04: Admin Analytics & Executive KPIs**.

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Master implementation plan for Feature 03 (Doctor Workspace & Shift Management). |
| **v1.1.0** | 2026-08-29 | Backend Lead | **Final Sign-Off Report** | Full execution report, 100% table-driven unit tests, real-time SSE stream validation, and live E2E integration test verification. |
