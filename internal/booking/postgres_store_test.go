package booking_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Xinnan-Alex/cinema/internal/adapters/postgres"
	"github.com/Xinnan-Alex/cinema/internal/booking"
	"github.com/google/uuid"
)

func setupTestStore(t *testing.T) *booking.PostgresStore {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool := postgres.NewPool(ctx, dbURL)
	t.Cleanup(func() { pool.Close() })

	if err := postgres.Migrate(ctx, pool, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Clean bookings between tests
	_, err := pool.Exec(ctx, "DELETE FROM bookings")
	if err != nil {
		t.Fatalf("clean bookings: %v", err)
	}

	return booking.NewPostgresStore(pool, 2*time.Minute)
}

func TestBook_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	b, err := store.Book(ctx, booking.Booking{
		MovieID: "test1",
		SeatID:  "A1",
		UserID:  "user-1",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if b.Status != "held" {
		t.Errorf("expected status held, got %s", b.Status)
	}
	if b.ID == "" {
		t.Error("expected non-empty booking ID")
	}
}

func TestBook_DuplicateSeatReturnsErrSeatTaken(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "A1", UserID: "user-1"})
	if err != nil {
		t.Fatalf("first book: %v", err)
	}

	_, err = store.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "A1", UserID: "user-2"})
	if err != booking.ErrSeatTaken {
		t.Errorf("expected ErrSeatTaken, got %v", err)
	}
}

func TestConfirm_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	held, _ := store.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "B1", UserID: "user-1"})

	confirmed, err := store.Confirm(ctx, held.ID, "user-1")
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != "confirmed" {
		t.Errorf("expected confirmed, got %s", confirmed.Status)
	}
}

func TestConfirm_WrongUser(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	held, _ := store.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "C1", UserID: "user-1"})

	_, err := store.Confirm(ctx, held.ID, "user-wrong")
	if err != booking.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestRelease_Success(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	held, _ := store.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "D1", UserID: "user-1"})

	err := store.Release(ctx, held.ID, "user-1")
	if err != nil {
		t.Fatalf("release: %v", err)
	}

	// Seat should be available again
	_, err = store.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "D1", UserID: "user-2"})
	if err != nil {
		t.Errorf("expected seat to be available after release, got %v", err)
	}
}

func TestListBookings_FiltersExpiredHolds(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Use a very short TTL store for this test
	dbURL := os.Getenv("TEST_DATABASE_URL")
	pool := postgres.NewPool(ctx, dbURL)
	defer pool.Close()
	shortStore := booking.NewPostgresStore(pool, 1*time.Millisecond)

	_, err := shortStore.Book(ctx, booking.Booking{MovieID: "test1", SeatID: "E1", UserID: "user-1"})
	if err != nil {
		t.Fatalf("book: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	bookings, err := store.ListBookings(ctx, "test1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, b := range bookings {
		if b.SeatID == "E1" {
			t.Error("expired hold should not appear in listing")
		}
	}
}

func TestConcurrentBooking_ExactlyOneWins(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	const numGoroutines = 1000
	seatID := fmt.Sprintf("RACE-%s", uuid.New().String()[:8])

	var (
		successes atomic.Int64
		failures  atomic.Int64
		wg        sync.WaitGroup
	)

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(n int) {
			defer wg.Done()
			_, err := store.Book(ctx, booking.Booking{
				MovieID: "test1",
				SeatID:  seatID,
				UserID:  fmt.Sprintf("user-%d", n),
			})
			if err == nil {
				successes.Add(1)
			} else {
				failures.Add(1)
			}
		}(i)
	}
	wg.Wait()

	if got := successes.Load(); got != 1 {
		t.Errorf("expected exactly 1 success, got %d", got)
	}
	if got := failures.Load(); got != int64(numGoroutines-1) {
		t.Errorf("expected %d failures, got %d", numGoroutines-1, got)
	}
}
