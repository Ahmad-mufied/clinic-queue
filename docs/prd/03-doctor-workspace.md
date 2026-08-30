# Feature Specification: Doctor Workspace & Consultation Workflow
**File:** `docs/prd/03-doctor-workspace.md`  
**Status:** Approved  
**Target Roles:** `doctor`

---

## 1. Feature Definition & Scope

The Doctor Workspace allows medical practitioners to manage their daily shift availability (`ONLINE` vs `OFFLINE`), call the next eligible patient from the queue, monitor elapsed consultation duration in real time, and complete consultations.

Actions in this workspace directly trigger **NATS JetStream events**, database updates, and SSE broadcasts to patients.

---

## 2. User Stories

1. **As a Doctor**, I want to toggle my status to `ONLINE` when I arrive at the clinic so that the queue engine includes me in the active doctor pool.
2. **As a Doctor**, I want to click "Call Next Patient" to admit the next patient in line and update their status.
3. **As a Doctor**, I want a live elapsed timer during consultation so I can monitor my pace against my average consultation target.
4. **As a Doctor**, I want to click "Finish Consultation" when done so that my room becomes available for the next patient.
5. **As a Doctor**, I want to toggle my status to `OFFLINE` during lunch breaks so that no new patients are routed to my room.

---

## 3. Case Scenarios

### 3.1 Positive Scenarios
- **[POS-DOC-01] Start Shift (Go Online):**  
  Doctor logs in and clicks `[Go Online]`. Doctor status becomes `is_online = true`. System emits `DOCTOR_SHIFT_STARTED` audit log and broadcasts updated doctor count to patients via SSE.
- **[POS-DOC-02] Call Next Patient:**  
  Doctor is idle and queue has 3 waiting patients. Doctor clicks `[Call Next Patient]`.  
  $\rightarrow$ Next patient (`A-01`, Alice) is popped from queue.  
  $\rightarrow$ Ticket status changes to `IN_CONSULTATION`.  
  $\rightarrow$ A new `consultation_sessions` record is created (`started_at = NOW()`).  
  $\rightarrow$ Live timer begins counting in Doctor UI ($00:00, 00:01\dots$).
- **[POS-DOC-03] Finish Active Consultation:**  
  Doctor finishes examining Alice after 3 minutes. Doctor clicks `[Finish Consultation]`.  
  $\rightarrow$ Session is closed (`finished_at = NOW()`, `is_active = false`).  
  $\rightarrow$ Ticket status becomes `COMPLETED`.  
  $\rightarrow$ Doctor room status resets to `IDLE / AVAILABLE`.  
  $\rightarrow$ Audit log `CONSULTATION_FINISHED` recorded with duration $3\text{m}$.  
  $\rightarrow$ SSE event triggers real-time queue recalculation for all waiting patients.

### 3.2 Negative Scenarios
- **[NEG-DOC-01] Call Next While Offline:**  
  Doctor attempts to call next patient while status is `OFFLINE`. System returns HTTP `400 Bad Request` (`"Doctor must be online to call patients"`).
- **[NEG-DOC-02] Call Next While Already in Consultation:**  
  Doctor who is currently seeing Patient A clicks "Call Next" again without finishing the current session. System returns HTTP `409 Conflict` (`"Active consultation already in progress. Finish current session first."`).
- **[NEG-DOC-03] Call Next When Queue is Empty:**  
  Doctor clicks "Call Next" when there are 0 patients waiting. System returns HTTP `200 OK` with payload `{ "message": "Queue is empty. No patients waiting." }` and no state mutation occurs.
- **[NEG-DOC-04] Finish Without Active Consultation:**  
  Doctor clicks "Finish" when room is idle. System returns HTTP `400 Bad Request` (`"No active consultation found to finish"`).

### 3.3 Edge Cases
- **[EDGE-DOC-01] Race Condition on "Call Next" (Multiple Doctors Call at the Same Millisecond):**  
  Doctor A and Doctor B click "Call Next" at the exact same millisecond when only 1 patient is in line.  
  $\rightarrow$ Handled atomically via PostgreSQL transaction / NATS JetStream WorkQueue consumer. Doctor A gets the patient; Doctor B receives notification that the queue is now empty.
- **[EDGE-DOC-02] Doctor Toggles Offline During Active Consultation:**  
  Doctor goes offline while examining a patient. The active consultation continues normally. Upon finishing, doctor transitions directly to `OFFLINE` instead of `IDLE`.
- **[EDGE-DOC-03] Extended Consultation (Overtime):**  
  Consultation exceeds target duration (e.g. 8 minutes on a 3-minute average). The timer displays in warning color (`⚠️ Overtime: 8m00s`). The engine clamps remaining time to $0\text{m}$ for waiting patients.

---

## 4. Acceptance Criteria & Identity Specification

### 4.1 Acceptance Criteria
- [ ] Doctors can only manipulate their own room and session state.
- [ ] Active session timer updates every second on the Doctor UI.
- [ ] Shift state transitions (`DOCTOR_SHIFT_STARTED`, `DOCTOR_SHIFT_ENDED`) are recorded with timestamps in `audit_logs`.
- [ ] Consultation completion immediately triggers NATS event and broadcasts recalculations via SSE.
- [ ] UI provides distinct visual states: `OFFLINE` (Gray), `IDLE / READY` (Green), `IN CONSULTATION` (Blue / Amber).

### 4.2 Identity & Identifier Separation (Database UUIDv7 vs Doctor Profile Name)
- **Database Identity (`id`, `doctor_id`, `session_id`):** 128-bit Native UUIDv7 string (e.g. `01919df4-8e3b-7412-a1f9-90b567c9e201`) for database relationships and consultation session keys.
- **Human Display Identity (`name` & `room`):** Formal practitioner name (`Dr. Sarah Adams`) and room label (`Room 1 - General Practice`) for consultation door plates, public doctor status monitors, and doctor workspace greetings.

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Doctor Workspace PRD. |
| **v1.1.0** | 2026-08-30 | Solution Architect | **Identity Design Standard** | Added Section 4.2 defining separation of internal UUIDv7 doctor/session IDs from human-facing practitioner names (`name: Dr. Sarah Adams`) and room headers. |
