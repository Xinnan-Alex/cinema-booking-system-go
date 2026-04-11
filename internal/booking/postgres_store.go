package booking

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool    *pgxpool.Pool
	holdTTL time.Duration
}

func NewPostgresStore(pool *pgxpool.Pool, holdTTL time.Duration) *PostgresStore {
	return &PostgresStore{pool: pool, holdTTL: holdTTL}
}

func (s *PostgresStore) Book(ctx context.Context, b Booking) (Booking, error) {
	id := uuid.New().String()
	expiresAt := time.Now().Add(s.holdTTL)

	// INSERT with ON CONFLICT DO NOTHING relies on the partial unique index
	// idx_active_seat (movie_id, seat_id) WHERE status IN ('held', 'confirmed').
	// If another active booking exists, zero rows are inserted.
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO bookings (id, movie_id, seat_id, user_id, status, expires_at)
		VALUES ($1, $2, $3, $4, 'held', $5)
		ON CONFLICT (movie_id, seat_id) WHERE status IN ('held', 'confirmed')
		DO NOTHING
	`, id, b.MovieID, b.SeatID, b.UserID, expiresAt)
	if err != nil {
		return Booking{}, err
	}

	if tag.RowsAffected() == 0 {
		return Booking{}, ErrSeatTaken
	}

	return Booking{
		ID:        id,
		MovieID:   b.MovieID,
		SeatID:    b.SeatID,
		UserID:    b.UserID,
		Status:    "held",
		ExpiresAt: expiresAt,
	}, nil
}

func (s *PostgresStore) ListBookings(ctx context.Context, movieID string) ([]Booking, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, movie_id, seat_id, user_id, status, expires_at
		FROM bookings
		WHERE movie_id = $1
		  AND status IN ('held', 'confirmed')
		  AND (status = 'confirmed' OR expires_at > now())
	`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []Booking
	for rows.Next() {
		var b Booking
		var expiresAt *time.Time
		if err := rows.Scan(&b.ID, &b.MovieID, &b.SeatID, &b.UserID, &b.Status, &expiresAt); err != nil {
			return nil, err
		}
		if expiresAt != nil {
			b.ExpiresAt = *expiresAt
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

func (s *PostgresStore) Confirm(ctx context.Context, sessionID string, userID string) (Booking, error) {
	var b Booking
	err := s.pool.QueryRow(ctx, `
		UPDATE bookings
		SET status = 'confirmed', expires_at = NULL, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND status = 'held' AND expires_at > now()
		RETURNING id, movie_id, seat_id, user_id, status
	`, sessionID, userID).Scan(&b.ID, &b.MovieID, &b.SeatID, &b.UserID, &b.Status)

	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrSessionNotFound
	}
	if err != nil {
		return Booking{}, err
	}
	return b, nil
}

func (s *PostgresStore) Release(ctx context.Context, sessionID string, userID string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM bookings
		WHERE id = $1 AND user_id = $2 AND status = 'held'
	`, sessionID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// CleanExpiredHolds removes held bookings that have passed their expiry.
// Run this periodically as a background goroutine.
func (s *PostgresStore) CleanExpiredHolds(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM bookings WHERE status = 'held' AND expires_at <= now()
	`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
