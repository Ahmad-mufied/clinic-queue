# Feature Specification: Executive Business Analytics & Doctor Productivity
**File:** `docs/prd/04-admin-analytics.md`  
**Status:** Approved  
**Target Roles:** `admin`

---

## 1. Feature Definition & Scope

The Executive Business Analytics Dashboard empowers clinic directors, operations managers, and executive leadership (CEO/COO) with data-driven insights into clinic throughput, operational bottlenecks, doctor productivity, and wait time accuracy.

It converts operational queue and session timestamps into actionable business intelligence metrics.

---

## 2. User Stories

1. **As a Clinic Director / CEO**, I want to see a high-level summary of total patients served, average waiting time, and queue depth today so that I can gauge clinic operational health at a glance.
2. **As an Operations Manager**, I want to view the **Doctor Productivity Table** (consultations completed, actual vs target duration, utilization rate) so that I can evaluate doctor performance and balance clinic workloads.
3. **As an Executive**, I want to compare **Estimated Wait Time vs Actual Wait Time** so that we can measure the accuracy of our AI/algorithmic prediction models.
4. **As an Operations Manager / Admin**, I want to view the **Hourly Patient Flow Distribution** (hourly check-ins, peak arrival periods, average intake per hour) so that I can anticipate staffing needs and manage lobby congestion.
5. **As an Admin**, I want to update doctor default average consultation times based on historical empirical data.

---

## 3. Mathematical Formulas & Business Metrics

### 3.1 Doctor Utilization Rate (%)
Measures how effectively a doctor's on-duty shift hours are utilized for actual patient consultations:
$$\text{Utilization Rate} = \left( \frac{\sum \text{Consultation Durations (Minutes)}}{\text{Total Working Hours Online (Minutes)}} \right) \times 100\%$$

### 3.2 Actual Waiting Time (Minutes)
The real elapsed duration from patient check-in to entering the consultation room:
$$\text{Actual Wait Time} = \text{called\_at} - \text{created\_at}$$

### 3.3 Consultation Duration Delta (Minutes)
The variance between empirical actual consultation time and configured target average:
$$\text{Duration Delta} = \text{Actual Duration} - \text{Target Avg Consultation Time}$$

---

## 4. Case Scenarios

### 4.1 Positive Scenarios
- **[POS-ADMIN-01] View Real-Time Executive KPI Cards:**  
  Admin opens Analytics Dashboard. System computes and displays:
  - Total Served: `42 Patients`
  - Current Queue Depth: `8 Waiting` | `2 In-Consultation`
  - Average Actual Wait Time: `14.2 Minutes`
  - Active Doctors: `2 / 2 Online`
- **[POS-ADMIN-02] Doctor Performance Breakdown:**  
  Admin reviews performance table:
  - Doctor A (Target 3m): 24 patients, Avg Actual 3.1m, Online 4h, Active 1.2h $\rightarrow$ Utilization **30.0%**.
  - Doctor B (Target 4m): 18 patients, Avg Actual 3.9m, Online 4h, Active 1.17h $\rightarrow$ Utilization **29.2%**.
- **[POS-ADMIN-03] Update Doctor Consultation Parameter:**  
  Admin updates Doctor A's configured average time from 3m to 4m. System persists change, emits `DOCTOR_CONFIG_UPDATED` audit log, and immediately updates live queue prediction calculations.

### 4.2 Negative Scenarios
- **[NEG-ADMIN-01] Unauthorized Role Access (Non-Admin):**  
  A patient or doctor attempts to request `GET /api/admin/stats`. Casbin middleware returns HTTP `403 Forbidden` (`"Administrator access required"`).
- **[NEG-ADMIN-02] Invalid Doctor Configuration Update:**  
  Admin attempts to set average consultation time to `<= 0` minutes. System returns HTTP `400 Bad Request` (`"Average consultation time must be greater than 0"`).

### 4.3 Edge Cases
- **[EDGE-ADMIN-01] Zero Patients Served (Start of Day / Empty Clinic):**  
  At 08:00 AM before any patient arrives: Total served = 0, Avg Wait = 0m, Utilization = 0%. System gracefully handles division by zero and displays clean zero states without errors.
- **[EDGE-ADMIN-02] Doctor Online for 0 Minutes (Just Logged In):**  
  Doctor went online 10 seconds ago. System avoids division by zero for utilization rate and displays `N/A (Shift Starting)`.
- **[EDGE-ADMIN-03] Overnight / Multi-Day Filter:**  
  Admin filters statistics by date range. Database queries filter by `created_at` timestamp range indexed on PostgreSQL 18.

---

## 5. Acceptance Criteria & Identity Specification

### 5.1 Acceptance Criteria
- [ ] Analytics queries execute in $< 50\text{ms}$ utilizing PostgreSQL indexes on timestamps and foreign keys.
- [ ] KPI cards, performance table, and hourly flow chart auto-refresh upon receiving SSE broadcast events.
- [ ] Admin dashboard UI features responsive charts and colored performance badges (Green = on target, Amber = slight delay, Red = significant bottleneck).
- [ ] Hourly patient flow chart visualizes dynamic admission time buckets (08:00 to 16:00) with automatic peak period highlighting.
- [ ] Changes to doctor configurations immediately reflect in new queue calculations.

### 5.2 Identity & Identifier Separation (Database UUIDv7 vs Doctor Performance Cards)
- **Database Identity (`doctor_id`):** 128-bit Native UUIDv7 string (e.g. `01919df4-8e3b-7412-a1f9-90b567c9e201`) for exact SQL GROUP BY aggregations and API update mutations (`POST /api/admin/doctors`).
- **Human Display Identity (`doctor_name`):** Human-readable practitioner name (`Dr. Sarah Adams`) rendered in executive leaderboard tables and KPI breakdowns.

---

## 6. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Solution Architect | **Initial Baseline** | Initial creation of the Executive Analytics PRD. |
| **v1.1.0** | 2026-08-30 | Solution Architect | **Identity Design Standard** | Added Section 5.2 defining separation of internal UUIDv7 doctor identifiers from executive leaderboard display names. |
| **v1.2.0** | 2026-08-30 | Solution Architect | **Hourly Flow Analytics** | Added Section 2 Story 4 and Section 5.1 Acceptance Criteria for dynamic Hourly Patient Flow Distribution & Peak Period detection. |
