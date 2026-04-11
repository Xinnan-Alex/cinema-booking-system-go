CREATE TABLE IF NOT EXISTS movies (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    rows        INT NOT NULL,
    seats_per_row INT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
