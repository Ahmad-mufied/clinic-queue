# Technical Specification: Executive Analytics & Business Intelligence
**File:** `docs/tech/04-admin-analytics-tech.md`  
**Status:** Approved  
**Version:** `v1.1.0`

---

## 1. Engineering Definition

The Executive Analytics subsystem performs high-performance SQL aggregations on PostgreSQL 18 to compute clinic KPIs, doctor utilization rates, and wait time variances for administrative decision-making.

---

## 2. Analytics SQL Queries & Metrics Implementation

### 2.1 Clinic Daily KPIs Aggregation

```sql
SELECT 
    COUNT(*) FILTER (WHERE status = 'COMPLETED') AS total_served_today,
    COUNT(*) FILTER (WHERE status = 'WAITING') AS current_waiting,
    COUNT(*) FILTER (WHERE status = 'IN_CONSULTATION') AS current_in_consultation,
    COALESCE(
        ROUND(AVG(EXTRACT(EPOCH FROM (called_at - created_at)) / 60.0)::numeric, 1), 
        0
    ) AS avg_actual_wait_minutes
FROM queue_tickets
WHERE created_at >= CURRENT_DATE;
```

### 2.2 Doctor Productivity & Utilization Rate Query

```sql
SELECT 
    d.id AS doctor_id,
    d.name AS doctor_name,
    d.avg_consultation_time_min AS target_avg_minutes,
    d.is_online,
    COUNT(cs.id) AS total_consultations_today,
    COALESCE(
        ROUND(AVG(EXTRACT(EPOCH FROM (cs.finished_at - cs.started_at)) / 60.0)::numeric, 1),
        0
    ) AS avg_actual_consultation_minutes,
    COALESCE(
        ROUND((SUM(EXTRACT(EPOCH FROM (cs.finished_at - cs.started_at)) / 60.0) / 
        NULLIF(EXTRACT(EPOCH FROM (NOW() - MIN(cs.started_at)) / 60.0), 0) * 100)::numeric, 1),
        0
    ) AS utilization_rate_percentage
FROM doctors d
LEFT JOIN consultation_sessions cs ON d.id = cs.doctor_id 
    AND cs.started_at >= CURRENT_DATE 
    AND cs.is_active = FALSE
GROUP BY d.id, d.name, d.avg_consultation_time_min, d.is_online
ORDER BY d.id ASC;
```

---

## 3. API Specification

### 3.1 Get Executive Analytics Summary
- **URL:** `GET /api/admin/stats`
- **Access:** Role `admin`
- **Response (200 OK):**
```json
{
  "summary": {
    "total_served_today": 42,
    "current_waiting": 8,
    "current_in_consultation": 2,
    "avg_actual_wait_minutes": 14.2,
    "online_doctors_count": 2
  },
  "doctor_performance": [
    {
      "doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e201",
      "doctor_name": "Doctor A",
      "target_avg_minutes": 3,
      "is_online": true,
      "total_consultations_today": 24,
      "avg_actual_consultation_minutes": 3.1,
      "utilization_rate_percentage": 62.0
    },
    {
      "doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e202",
      "doctor_name": "Doctor B",
      "target_avg_minutes": 4,
      "is_online": true,
      "total_consultations_today": 18,
      "avg_actual_consultation_minutes": 3.9,
      "utilization_rate_percentage": 58.5
    }
  ]
}
```

### 3.2 Update Doctor Configuration
- **URL:** `POST /api/admin/doctors`
- **Access:** Role `admin`
- **Request Body:**
```json
{
  "doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e201",
  "avg_consultation_time_min": 4
}
```
- **Response (200 OK):** Updated doctor configuration object.

---

## 4. API Case Scenarios

| Scenario ID | Endpoint | Method | Condition | Status | Response Summary |
| :--- | :--- | :---: | :--- | :---: | :--- |
| **API-ADM-01** | `/api/admin/stats` | `GET` | Valid Admin Token | `200 OK` | Returns full KPI summary & doctor table with UUIDv7 IDs |
| **API-ADM-02** | `/api/admin/stats` | `GET` | Zero patients today | `200 OK` | Returns safe zero values without errors |
| **API-ADM-03** | `/api/admin/stats` | `GET` | Patient/Doctor Token | `403 Forbidden` | `{"error": "Access denied: insufficient privileges"}` |
| **API-ADM-04** | `/api/admin/doctors`| `POST`| `avg_time: 0` | `400 Bad Request` | `{"error": "Avg time must be greater than 0"}` |

---

## 5. Document Revision History & Requirement Changelog

| Version | Date | Author / Role | Change Type | Change Summary / Rationale |
| :---: | :---: | :---: | :---: | :--- |
| **v1.0.0** | 2026-08-29 | Backend Lead | **Initial Baseline** | Initial technical specification for PostgreSQL 18 analytics queries, doctor utilization rate calculation, division-by-zero safeguards, and executive REST endpoints. |
| **v1.1.0** | 2026-08-30 | Backend Lead | **Native UUIDv7 Spec** | Migrated `DoctorPerformance.DoctorID` and `UpdateDoctorConfigRequest.DoctorID` to Native UUIDv7 string identifiers. |
