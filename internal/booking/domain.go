package booking

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSeatTaken        = errors.New("seat already taken")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
	ErrUnauthorized     = errors.New("unauthorized")
)

type Booking struct {
	ID        string
	MovieID   string
	SeatID    string
	UserID    string
	Status    string
	ExpiresAt time.Time
}

type BookingStore interface {
	Book(ctx context.Context, b Booking) (Booking, error)
	ListBookings(ctx context.Context, movieID string) ([]Booking, error)
	Release(ctx context.Context, sessionID string, userID string) error
	Confirm(ctx context.Context, sessionID string, userID string) (Booking, error)
}
