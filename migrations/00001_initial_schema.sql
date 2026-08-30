-- +goose Up
CREATE TABLE IF NOT EXISTS doctors (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(100) NOT NULL,
    avg_consultation_time_min INT NOT NULL CHECK (avg_consultation_time_min > 0),
    is_online BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('patient', 'doctor', 'admin')),
    doctor_id UUID REFERENCES doctors(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

CREATE TABLE IF NOT EXISTS queue_tickets (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    patient_name VARCHAR(100) NOT NULL,
    queue_number VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'WAITING' 
        CHECK (status IN ('WAITING', 'IN_CONSULTATION', 'COMPLETED', 'CANCELLED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    called_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_queue_tickets_status ON queue_tickets(status);
CREATE INDEX IF NOT EXISTS idx_queue_tickets_created_at ON queue_tickets(created_at);

CREATE TABLE IF NOT EXISTS consultation_sessions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    doctor_id UUID NOT NULL REFERENCES doctors(id) ON DELETE CASCADE,
    ticket_id UUID NOT NULL REFERENCES queue_tickets(id) ON DELETE CASCADE,
    patient_name VARCHAR(100) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX IF NOT EXISTS idx_sessions_doctor_active ON consultation_sessions(doctor_id, is_active);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB DEFAULT '{}'::jsonb,
    ip_address VARCHAR(45),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_created ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_details_gin ON audit_logs USING GIN (details);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS consultation_sessions;
DROP TABLE IF EXISTS queue_tickets;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS doctors;
