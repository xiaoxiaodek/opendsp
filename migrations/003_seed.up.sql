-- Seed initial admin user (password: admin123)
INSERT INTO users (email, password_hash, name, role) 
VALUES ('admin@opendsp.io', '240be518fabd2724ddb6f04eeb1da5967448d7e831c08c8fa822809f74c720a9', 'Super Admin', 'admin')
ON CONFLICT (email) DO NOTHING;
