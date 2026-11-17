-- Migration: 003_create_bookings_and_logs_table.sql
-- Description: Create bookings and booking_logs tables for CampusSpace Backend

CREATE TABLE IF NOT EXISTS bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    room_id BIGINT NOT NULL REFERENCES rooms(id),
    purpose TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_booking_time CHECK (start_time < end_time)
);

-- Index untuk mempercepat query relasi dan status booking
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings (user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_room_id ON bookings (room_id);
CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings (status);

CREATE TABLE IF NOT EXISTS booking_logs (
    id BIGSERIAL PRIMARY KEY,
    booking_id BIGINT NOT NULL REFERENCES bookings(id),
    actor_id BIGINT NOT NULL REFERENCES users(id),
    from_status VARCHAR(20) NULL CHECK (from_status IS NULL OR from_status IN ('pending', 'approved', 'rejected', 'cancelled')),
    to_status VARCHAR(20) NOT NULL CHECK (to_status IN ('pending', 'approved', 'rejected', 'cancelled')),
    note TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index untuk mempercepat query audit log berdasarkan booking dan actor
CREATE INDEX IF NOT EXISTS idx_booking_logs_booking_id ON booking_logs (booking_id);
CREATE INDEX IF NOT EXISTS idx_booking_logs_actor_id ON booking_logs (actor_id);
