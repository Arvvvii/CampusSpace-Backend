-- Migration: 001_create_rooms_table.sql
-- Description: Create rooms table for CampusSpace Backend

CREATE TABLE IF NOT EXISTS rooms (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    building VARCHAR(100) NOT NULL,
    capacity INTEGER NOT NULL CHECK (capacity > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index pada kolom is_active untuk mengoptimalkan pencarian ruangan aktif
CREATE INDEX IF NOT EXISTS idx_rooms_is_active ON rooms (is_active);
