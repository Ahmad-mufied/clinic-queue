-- +goose Up
-- Seed Doctors
INSERT INTO doctors (id, name, avg_consultation_time_min, is_online)
VALUES 
    (1, 'Doctor A', 3, false),
    (2, 'Doctor B', 4, false)
ON CONFLICT (id) DO NOTHING;

SELECT setval('doctors_id_seq', (SELECT MAX(id) FROM doctors));

-- Seed Demo Users (password is 'password123' for all demo accounts)
-- bcrypt hash: $2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.
INSERT INTO users (username, password_hash, name, role, doctor_id)
VALUES
    ('doctor_a', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Dr. Sarah Adams', 'doctor', 1),
    ('doctor_b', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Dr. Michael Chen', 'doctor', 2),
    ('patient_john', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'John Doe', 'patient', NULL),
    ('patient_lucas', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Lucas Smith', 'patient', NULL),
    ('admin', '$2a$10$LtdYqpQiGxxNVRcHAVEIU.ehPKtqxNt8g1yaK2YltTbhBLPbst2H.', 'Clinic Administrator', 'admin', NULL)
ON CONFLICT (username) DO NOTHING;

-- +goose Down
DELETE FROM users WHERE username IN ('doctor_a', 'doctor_b', 'patient_john', 'patient_lucas', 'admin');
DELETE FROM doctors WHERE id IN (1, 2);
