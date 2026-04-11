package booking

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSeatTaken = errors.New("seat taken")
)

// Booking represents a confirmed seat reservation.
type Booking struct {
	ID        string
	MovieID   string
	SeatID    string
	UserID    string
	Status    string
	ExpiresAt time.Time
}

type BookingStore interface {
	Book(b Booking) (Booking, error)
	ListBookings(movieID string) []Booking
	Release(ctx context.Context, sessionID string, userID string) error
	Confirm(ctx context.Context, sessionID string, userID string) (Booking, error)
}
