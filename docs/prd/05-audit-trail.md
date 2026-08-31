# Feature Specification: Comprehensive Activity Logging & Audit Trail
**File:** `docs/prd/05-audit-trail.md`  
**Status:** Approved  
**Target Roles:** `admin`

---

## 1. Feature Definition & Scope

The Audit Trail subsystem captures an immutable, chronological stream of all security, operational, and lifecycle events across the clinic platform. 

It satisfies healthcare compliance standards, enables forensic accountability, and serves as the data foundation for business analytics.

---

## 2. User Stories

1. **As a Security & Compliance Officer**, I want every login, logout, and access attempt recorded with timestamps, user ID, and IP address so that unauthorized access can be investigated.
2. **As an Operations Auditor**, I want to see the full historical lifecycle of any queue ticket (when joined, who called it, when completed) to resolve patient complaints or disputes.
3. **As a Clinic Administrator**, I want a paginated and filterable Audit Log Feed in the UI so that I can inspect recent clinic activities in real time.

---

## 3. Audit Event Taxonomy

| Event Category | Action Identifier | Actor | Details Payload Schema (JSONB) |
| :--- | :--- | :--- | :--- |
| **Authentication** | `AUTH_LOGIN` | Any | `{ "username": "dr_smith", "role": "doctor", "success": true }` |
| | `AUTH_LOGOUT` | Any | `{ "session_duration_min": 120 }` |
| | `AUTH_REGISTER` | Patient | `{ "username": "john_doe", "name": "John Doe" }` |
| **Doctor Shift** | `DOCTOR_SHIFT_STARTED` | Doctor | `{ "doctor_id": 1, "doctor_name": "Doctor A" }` |
| | `DOCTOR_SHIFT_ENDED` | Doctor | `{ "doctor_id": 1, "shift_duration_min": 240 }` |
| **Queue Lifecycle** | `QUEUE_JOINED` | Patient | `{ "ticket_id": 101, "queue_number": "A-06", "ahead_count": 5 }` |
| | `CONSULTATION_STARTED` | Doctor | `{ "ticket_id": 101, "doctor_id": 1, "patient": "John" }` |
| | `CONSULTATION_FINISHED`| Doctor | `{ "ticket_id": 101, "actual_duration_min": 3.2, "target_min": 3 }` |
| | `QUEUE_CANCELLED` | Patient / Admin| `{ "ticket_id": 101, "reason": "patient_left" }` |
| **Configuration** | `DOCTOR_CONFIG_UPDATED`| Admin | `{ "doctor_id": 1, "old_avg_min": 3, "new_avg_min": 4 }` |

---

## 4. Case Scenarios

### 4.1 Positive Scenarios
- **[POS-AUDIT-01] Automated Queue Event Logging:**  
  When John clicks "Join Queue", system creates ticket in DB and asynchronously captures the event via NATS JetStream `AuditWorker`, inserting an `audit_logs` record (`action = 'QUEUE_JOINED'`, actor John, IP `127.0.0.1`).
- **[POS-AUDIT-02] Doctor Consultation Lifecycle Logging:**  
  When Doctor A calls John $\rightarrow$ `CONSULTATION_STARTED` logged. When Doctor A completes examination after $3.5\text{m}$ $\rightarrow$ `CONSULTATION_FINISHED` logged with `details = {"actual_duration_minutes": 3.5, "delta": +0.5}`.
- **[POS-AUDIT-03] Filter Audit Logs by Actor, Role, and Action:**  
  Admin searches for events filtered by action `"CONSULTATION_FINISHED"` or role `"doctor"`. System returns filtered results using high-performance cursor pagination in $< 10\text{ms}$.
- **[POS-AUDIT-04] Live Audit Stream (SSE) & Auto Infinite Scroll:**  
  Admin stays on the Audit Log screen. As new clinic actions occur in consultation rooms, new log rows append to the top of the table in real time via SSE. When the admin scrolls inside the table container, older logs auto-load seamlessly without pagination drift or duplicate records.

### 4.2 Negative Scenarios
- **[NEG-AUDIT-01] Non-Admin Audit Log Access:**  
  Patient or Doctor attempts to query `GET /api/admin/audit-logs`. System returns HTTP `403 Forbidden`.
- **[NEG-AUDIT-02] Audit Log Mutation Attempt:**  
  No API endpoint exists to update (`PUT/PATCH`) or delete (`DELETE`) audit records. The `audit_logs` table is strictly append-only.

### 4.3 Edge Cases
- **[EDGE-AUDIT-01] High-Concurrency Batch Events:**  
  100 patients join during morning rush. System handles asynchronous log ingestion via decoupled NATS JetStream `AuditWorker` without blocking HTTP request latency.
- **[EDGE-AUDIT-02] Unauthenticated / Public Events:**  
  A public user accesses the live board or fails a login attempt. System records `user_id = NULL`, `actor_name = 'Anonymous / System'`, and stores client IP.

---

## 5. Acceptance Criteria & Identity Specification

### 5.1 Acceptance Criteria
- [x] Every state-changing API request produces an audit log entry via decoupled NATS JetStream `AuditWorker`.
- [x] Audit logs table includes `created_at`, `user_id`, `actor_name`, `role`, `action`, `details` (JSONB), and `ip_address`.
- [x] Client forensic metadata (`ip_address`, `user_agent`, `request_id`) is captured via HTTP middleware context propagation and stored in `audit_logs` (with `user_agent` and `request_id` preserved in JSONB `details`).
- [x] Log entries are strictly immutable (append-only) with index on `id DESC` and `(action, created_at DESC)`.
- [x] Backend API provides high-efficiency Cursor Pagination (`cursor`, `limit`, `next_cursor`, `has_more`) to eliminate pagination drift.
- [x] Advanced Filtering: Supports keyword search across Actor, Action, IP address, and Date Range (`start_date`, `end_date`).
- [x] Bidirectional Sorting: Supports `sort_order` parameter (`desc` for Newest First, `asc` for Oldest First) with bidirectional cursor pagination.
- [x] Admin UI provides filter toolbar (Search, Date Range, Sort Order, Action, Role), JSON payload inspector modal, and contained table scrolling with sticky header.
- [x] Automatic infinite scroll auto-fetches records on scroll threshold without manual button clicks.
- [x] Live audit stream broadcasts new events to admin subscribers via SSE with sub-second table synchronization.

### 5.2 Identity & Identifier Separation (Database UUIDv7 vs Human Actor Badges)
- **Database Identity (`id`, `user_id`):** 128-bit Native UUIDv7 string (e.g. `01919df4-8e3b-7412-a1f9-90b567c9e536`) for immutable primary keys, cursor comparison (`WHERE id < $cursor`), and relational integrity.
- **Human Display Identity (`actor_name` & `role`):** Clear user badges (`Dr. Michael Chen (@doctor_b)`, `Patient Lucas`, `Clinic Admin (@admin)`) displayed directly in the UI table feed and inspector modal overview cards, while preserving full UUIDv7 keys in the forensic JSON payload viewer.

---

## 6. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Comprehensive Activity Logging PRD. |
| **v1.1.0** | 2026-08-30 | Lead Solution Architect | **Architecture Enhancement** | Upgraded to Cursor-Based Infinite Lazy Loading (`id < cursor`), decoupled NATS JetStream async `AuditWorker` ingestion, internal table container scrolling with sticky header, and auto-fetch on scroll threshold. |
| **v1.2.0** | 2026-08-30 | Lead Solution Architect | **Feature Enhancement** | Added keyword search (Actor, Action, IP), Date Range filters, and Bidirectional Sorting (ASC/DESC) to the Audit Trail pipeline. |
| **v1.3.0** | 2026-08-30 | Lead Solution Architect | **Identity Design Standard** | Added Section 5.2 defining separation of internal UUIDv7 audit IDs and user IDs from human-friendly actor badges and usernames in UI activity feeds. |
| **v1.4.0** | 2026-08-31 | Backend Security Engineer | **Forensic Metadata Spec** | Added Section 5.1 acceptance criteria for client forensic metadata propagation (`ip_address`, `user_agent`, `request_id`) across Context, NATS, and Audit Worker. |
