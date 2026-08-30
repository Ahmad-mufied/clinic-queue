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
ON CONFLICT (username) DO NOTHING;

-- +goose Down
DELETE FROM users WHERE username IN ('doctor_a', 'doctor_b', 'patient_john', 'patient_lucas', 'admin');
DELETE FROM doctors WHERE id IN ('01919df4-8e3b-7412-a1f9-90b567c9e101', '01919df4-8e3b-7412-a1f9-90b567c9e102');
