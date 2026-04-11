CREATE TABLE IF NOT EXISTS bookings (
    id          UUID PRIMARY KEY,
    movie_id    TEXT NOT NULL REFERENCES movies(id),
    seat_id     TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'held' CHECK (status IN ('held', 'confirmed')),
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Only one active (held or confirmed) booking per seat per movie.
-- This is the PostgreSQL equivalent of Redis SET NX for double-booking prevention.
CREATE UNIQUE INDEX IF NOT EXISTS idx_active_seat
    ON bookings (movie_id, seat_id)
    WHERE status IN ('held', 'confirmed');
