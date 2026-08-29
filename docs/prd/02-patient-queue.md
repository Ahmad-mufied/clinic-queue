# Feature Specification: Patient Queue & Real-Time Wait Estimation
**File:** `docs/prd/02-patient-queue.md`  
**Status:** Approved  
**Target Roles:** `patient`, `public`

---

## 1. Feature Definition & Scope

This feature enables walk-in patients to register online for the daily clinic queue, receive a digital queue ticket, and monitor their live estimated waiting time. 

The waiting time is computed dynamically by the **Greedy Multi-Doctor Queue Engine** and streams live updates via **Server-Sent Events (SSE)** whenever doctor availability, active consultations, or queue positions change.

---

## 2. User Stories

1. **As a Walk-in Patient**, I want to submit my name to join the queue so that I secure my place in line without waiting physically in the clinic lobby.
2. **As a Waiting Patient**, I want to see a live estimated wait time in minutes so that I can manage my schedule accurately.
3. **As a Waiting Patient**, I want my ticket status to update automatically when a doctor calls my number without needing to refresh my browser.
4. **As a Patient**, I want to know if all doctors are currently offline so that I understand why the countdown is paused.

---

## 3. Case Scenarios

### 3.1 Positive Scenarios
- **[POS-QUEUE-01] Patient Joins Queue (Single Idle Doctor - Case Study 1 Q1):**  
  Doctor 1 has 3m average time (idle). 5 patients are ahead. John joins as 6th.  
  $\rightarrow$ System creates ticket `A-06` with status `WAITING`. Engine calculates: $5 \times 3\text{m} = \mathbf{15\text{ mins}}$.
- **[POS-QUEUE-02] Patient Joins Queue (Multi-Doctor with Active Session - Case Study 2 Q2):**  
  Doctor A is idle ($0\text{m}$), Doctor B has Lucas in consultation ($2\text{m}$ remaining). 9 patients are waiting. John joins as 10th waiting in line (11th overall).  
  $\rightarrow$ Engine simulates greedy dispatch across Doctor A & B $\rightarrow$ Returns John's wait time: **15 minutes** (Doctor A).
- **[POS-QUEUE-03] Immediate Entry (1st in Queue):**  
  A patient arrives when all doctors are idle and no one is in queue. Patient joins $\rightarrow$ Estimated Wait Time: **0 minutes** (ready for immediate call).
- **[POS-QUEUE-04] Real-time SSE Broadcast on State Change:**  
  When Doctor A finishes a patient and calls the next, an SSE event `QUEUE_UPDATED` is pushed to all connected patients. John's app updates his ahead count from 5 to 4, and his wait time recalculates automatically.

### 3.2 Negative Scenarios
- **[NEG-QUEUE-01] Empty or Invalid Patient Name:**  
  Patient submits queue form with empty name or only whitespace. System returns HTTP `400 Bad Request` (`"Patient name is required"`).
- **[NEG-QUEUE-02] Duplicate Active Queue Entry:**  
  A patient who already has a ticket with status `WAITING` or `IN_CONSULTATION` attempts to join again. System returns HTTP `409 Conflict` (`"You already have an active ticket in the queue"`).
- **[NEG-QUEUE-03] Joining Queue When Clinic Is Closed / Zero Doctors Exist:**  
  Patient attempts to join when no doctors are registered in the system. System returns HTTP `503 Service Unavailable` (`"No doctors currently configured for this clinic"`).

### 3.3 Edge Cases
- **[EDGE-QUEUE-01] All Doctors Offline / On Break:**  
  All doctors are toggled to `OFFLINE`. The API returns `estimated_wait_time_minutes: null` with `notice: "Estimated wait time is currently unavailable because all doctors are offline / on break. Calculation will activate once a doctor starts duty."`. Patient's UI displays the notice and pauses countdown cleanly without displaying misleading 0-minute countdowns.
- **[EDGE-QUEUE-02] Doctor Consultation Exceeds Average Time (Peter 5m in 3m Avg - CS1 Q2b):**  
  Doctor 1 has 3m average, but Peter's elapsed time is 5m ($5 > 3$). The engine automatically computes $\text{RemainingTime} = \max(0, 3 - 5) = \mathbf{0\text{ minutes}}$ (assumes completion momentarily), preventing negative wait times.
- **[EDGE-QUEUE-03] Doctor Unexpectedly Goes Offline While Patients Are Waiting:**  
  Doctor B goes offline while John is waiting. The engine immediately recomputes queue distribution using only remaining online doctors (Doctor A) and broadcasts the updated wait time to John via SSE.
- **[EDGE-QUEUE-04] Patient Drops Out / Cancels Ticket:**  
  Patient in position 3 cancels ticket. NATS stream and DB update ticket to `CANCELLED`. All patients behind position 3 instantly see their ahead count decrement and wait times adjust.

---

## 4. Acceptance Criteria

- [ ] Queue numbers follow an incremental formatted sequence (e.g., `A-01`, `A-02`).
- [ ] Estimated wait time algorithm matches verified mathematical models (Case Study 1 & 2 verified outputs).
- [ ] SSE updates are received by all connected clients in $< 100\text{ms}$ after any queue event.
- [ ] When called by a doctor, patient's screen highlights with room assignment (e.g., *"Please enter Doctor A's Room"*).
- [ ] Ticket lifecycle transitions (`QUEUE_JOINED`, `QUEUE_CANCELLED`) are persisted to `audit_logs`.

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Patient Queue PRD, incorporating Case Study 1 & 2 mathematical queue models, real-time SSE wait countdowns, positive/negative/edge scenarios, and offline doctor handling. |
