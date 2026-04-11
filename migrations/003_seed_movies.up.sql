INSERT INTO movies (id, title, rows, seats_per_row) VALUES
    ('test1', 'test1', 5, 8),
    ('test2', 'test2', 4, 6)
ON CONFLICT (id) DO NOTHING;
