-- +goose Up
-- Seed Doctors
INSERT INTO doctors (id, name, avg_consultation_time_min, is_online)
VALUES 
    ('01919df4-8e3b-7412-a1f9-90b567c9e101', 'Dr. Sarah Adams', 3, true),
    ('01919df4-8e3b-7412-a1f9-90b567c9e102', 'Dr. Michael Chen', 4, true)
ON CONFLICT (id) DO NOTHING;

-- Seed Demo Users (password is 'password123' for all demo accounts)
-- bcrypt hash: $2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.
INSERT INTO users (id, username, password_hash, name, role, doctor_id)
VALUES
    ('01919df4-8e3b-7412-a1f9-90b567c9e201', 'doctor_a', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Dr. Sarah Adams', 'doctor', '01919df4-8e3b-7412-a1f9-90b567c9e101'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e202', 'doctor_b', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Dr. Michael Chen', 'doctor', '01919df4-8e3b-7412-a1f9-90b567c9e102'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e203', 'patient_john', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'John Doe', 'patient', NULL),
    ('01919df4-8e3b-7412-a1f9-90b567c9e204', 'patient_lucas', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Lucas Smith', 'patient', NULL),
    ('01919df4-8e3b-7412-a1f9-90b567c9e205', 'admin', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Clinic Administrator', 'admin', NULL)
-- Seed Demo Audit Logs
INSERT INTO audit_logs (id, user_id, actor_name, role, action, details, ip_address, created_at)
VALUES
    ('01919df4-8e3b-7412-a1f9-90b567c9e301', '01919df4-8e3b-7412-a1f9-90b567c9e205', 'Clinic Administrator', 'admin', 'DOCTOR_CONFIG_UPDATED', '{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "avg_time_min": 3, "reason": "Standard Morning Configuration"}'::jsonb, '127.0.0.1', NOW() - INTERVAL '45 minutes'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e302', '01919df4-8e3b-7412-a1f9-90b567c9e201', 'Dr. Sarah Adams', 'doctor', 'DOCTOR_STATUS_CHANGED', '{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "is_online": true, "status": "AVAILABLE"}'::jsonb, '127.0.0.1', NOW() - INTERVAL '40 minutes'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e303', '01919df4-8e3b-7412-a1f9-90b567c9e202', 'Dr. Michael Chen', 'doctor', 'DOCTOR_STATUS_CHANGED', '{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e102", "is_online": true, "status": "AVAILABLE"}'::jsonb, '127.0.0.1', NOW() - INTERVAL '35 minutes'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e304', '01919df4-8e3b-7412-a1f9-90b567c9e203', 'John Doe', 'patient', 'QUEUE_JOINED', '{"ticket_id": "01919df4-8e3b-7412-a1f9-90b567c9e401", "queue_number": "A-01", "estimated_wait_minutes": 0}'::jsonb, '127.0.0.1', NOW() - INTERVAL '30 minutes'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e305', '01919df4-8e3b-7412-a1f9-90b567c9e204', 'Lucas Smith', 'patient', 'QUEUE_JOINED', '{"ticket_id": "01919df4-8e3b-7412-a1f9-90b567c9e402", "queue_number": "A-02", "estimated_wait_minutes": 0}'::jsonb, '127.0.0.1', NOW() - INTERVAL '25 minutes'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e306', NULL, 'Dr. Sarah Adams', 'doctor', 'CONSULTATION_STARTED', '{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "ticket_number": "A-01", "patient_name": "John Doe"}'::jsonb, '127.0.0.1', NOW() - INTERVAL '20 minutes'),
    ('01919df4-8e3b-7412-a1f9-90b567c9e307', NULL, 'Dr. Sarah Adams', 'doctor', 'CONSULTATION_FINISHED', '{"doctor_id": "01919df4-8e3b-7412-a1f9-90b567c9e101", "ticket_number": "A-01", "duration_minutes": 4}'::jsonb, '127.0.0.1', NOW() - INTERVAL '15 minutes')
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM audit_logs WHERE id IN (
    '01919df4-8e3b-7412-a1f9-90b567c9e301',
    '01919df4-8e3b-7412-a1f9-90b567c9e302',
    '01919df4-8e3b-7412-a1f9-90b567c9e303',
    '01919df4-8e3b-7412-a1f9-90b567c9e304',
    '01919df4-8e3b-7412-a1f9-90b567c9e305',
    '01919df4-8e3b-7412-a1f9-90b567c9e306',
    '01919df4-8e3b-7412-a1f9-90b567c9e307'
);
DELETE FROM users WHERE username IN ('doctor_a', 'doctor_b', 'patient_john', 'patient_lucas', 'admin');
DELETE FROM doctors WHERE id IN ('01919df4-8e3b-7412-a1f9-90b567c9e101', '01919df4-8e3b-7412-a1f9-90b567c9e102');
