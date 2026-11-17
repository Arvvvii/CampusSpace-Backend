-- Migration: 002_create_users_table.sql
-- Description: Create users table for CampusSpace Backend

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('student', 'approver', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index pada kolom role untuk pencarian berdasarkan hak akses
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
