export type Role = "patient" | "doctor" | "admin";

export interface User {
  id: string;
  username: string;
  name: string;
  role: Role;
  doctor_id?: string | null;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface QueueTicket {
  id: string;
  user_id?: string | null;
  patient_name: string;
  queue_number: string;
  status: "WAITING" | "IN_CONSULTATION" | "COMPLETED" | "CANCELLED";
  position_in_queue?: number;
  ahead_count?: number;
  position?: number;
  estimated_wait_time_minutes?: number | null;
  notice?: string;
  created_at: string;
  called_at?: string;
  finished_at?: string;
}

export interface DoctorAvailability {
  id: string;
  name: string;
  avg_time: number;
  is_online: boolean;
  status: "AVAILABLE" | "IN_CONSULTATION" | "OFFLINE";
  current_patient?: string;
  elapsed_minutes?: number;
}

export interface QueueTicketSummary {
  queue_number: string;
  patient_name: string;
  estimated_wait_minutes?: number | null;
  notice?: string;
}

export interface QueueStatusResponse {
  online_doctors: DoctorAvailability[];
  total_waiting: number;
  queue_list: QueueTicketSummary[];
  notice?: string;
  online_doctors_count?: number;
  waiting_tickets?: QueueTicket[];
  estimated_wait_time_minutes?: number;
}

export interface ConsultationSession {
  id: string;
  doctor_id: string;
  ticket_id: string;
  patient_name: string;
  started_at: string;
  ended_at?: string | null;
  actual_duration_seconds?: number | null;
  is_active: boolean;
  ticket?: QueueTicket;
}

export interface DoctorWorkspace {
  doctor_id: string;
  doctor_name: string;
  avg_consultation_time?: number;
  is_online: boolean;
  status: "OFFLINE" | "AVAILABLE" | "IN_CONSULTATION";
  active_session?: ConsultationSession | null;
}

export interface DoctorPerformance {
  doctor_id: string;
  doctor_name: string;
  username?: string;
  target_avg_minutes: number;
  is_online: boolean;
  total_consultations_today: number;
  avg_actual_consultation_minutes: number;
  utilization_rate_percentage: number;
}

export interface HourlyPatientFlow {
  hour_label: string;
  patient_count: number;
  height_percentage: number;
  is_peak: boolean;
}

export interface AdminDashboardStats {
  summary: {
    total_served_today: number;
    avg_waiting_time_minutes: number;
    online_doctors_count: number;
    clinic_utilization_rate_percentage: number;
  };
  doctor_performance: DoctorPerformance[];
  hourly_distribution?: HourlyPatientFlow[];
}

export interface AuditLog {
  id: string;
  action: string;
  user_id?: string | null;
  actor_name: string;
  role: string;
  ip_address: string;
  details?: Record<string, any>;
  metadata?: Record<string, any>;
  created_at: string;
}

export interface AuditLogParams {
  page?: number;
  limit?: number;
  cursor?: string | null;
  search?: string;
  action?: string;
  role?: string;
  user_id?: string;
  start_date?: string;
  end_date?: string;
  sort_order?: "desc" | "asc";
}

export interface PaginatedAuditLogs {
  page?: number;
  limit: number;
  next_cursor?: string | null;
  has_more?: boolean;
  total_records: number;
  total_pages: number;
  logs: AuditLog[];
}

export interface SSEEventPayload {
  type: "QUEUE_UPDATED" | "TICKET_CALLED" | "TICKET_FINISHED" | "DOCTOR_STATUS_CHANGED" | "DOCTOR_CONFIG_UPDATED" | "AUDIT_LOG_CREATED";
  timestamp: string;
  data: Record<string, any>;
}

export interface DemoPersona {
  id: string;
  label: string;
  role: Role;
  username: string;
  password: string;
  name: string;
  description: string;
  doctor_id?: string;
}
