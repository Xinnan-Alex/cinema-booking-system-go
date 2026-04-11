package booking_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Xinnan-Alex/cinema/internal/booking"
)

// mockStore implements booking.BookingStore for handler tests.
type mockStore struct {
	bookFn    func(ctx context.Context, b booking.Booking) (booking.Booking, error)
	listFn    func(ctx context.Context, movieID string) ([]booking.Booking, error)
	confirmFn func(ctx context.Context, sessionID, userID string) (booking.Booking, error)
	releaseFn func(ctx context.Context, sessionID, userID string) error
}

func (m *mockStore) Book(ctx context.Context, b booking.Booking) (booking.Booking, error) {
	return m.bookFn(ctx, b)
}
func (m *mockStore) ListBookings(ctx context.Context, movieID string) ([]booking.Booking, error) {
	return m.listFn(ctx, movieID)
}
func (m *mockStore) Confirm(ctx context.Context, sessionID, userID string) (booking.Booking, error) {
	return m.confirmFn(ctx, sessionID, userID)
}
func (m *mockStore) Release(ctx context.Context, sessionID, userID string) error {
	return m.releaseFn(ctx, sessionID, userID)
}

func TestHoldSeat_Success(t *testing.T) {
	store := &mockStore{
		bookFn: func(_ context.Context, b booking.Booking) (booking.Booking, error) {
			return booking.Booking{
				ID:        "session-123",
				MovieID:   b.MovieID,
				SeatID:    b.SeatID,
				UserID:    b.UserID,
				Status:    "held",
				ExpiresAt: time.Now().Add(2 * time.Minute),
			}, nil
		},
	}
	svc := booking.NewService(store)
	h := booking.NewHandler(svc)

	body := `{"userID":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/movies/test1/seats/A1/hold", strings.NewReader(body))
	req.SetPathValue("movieID", "test1")
	req.SetPathValue("seatID", "A1")
	w := httptest.NewRecorder()

	h.HoldSeat(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["sessionID"] != "session-123" {
		t.Errorf("expected session-123, got %s", resp["sessionID"])
	}
}

func TestHoldSeat_Conflict(t *testing.T) {
	store := &mockStore{
		bookFn: func(_ context.Context, _ booking.Booking) (booking.Booking, error) {
			return booking.Booking{}, booking.ErrSeatTaken
		},
	}
	svc := booking.NewService(store)
	h := booking.NewHandler(svc)

	body := `{"userID":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/movies/test1/seats/A1/hold", strings.NewReader(body))
	req.SetPathValue("movieID", "test1")
	req.SetPathValue("seatID", "A1")
	w := httptest.NewRecorder()

	h.HoldSeat(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestHoldSeat_MissingUserID(t *testing.T) {
	svc := booking.NewService(&mockStore{})
	h := booking.NewHandler(svc)

	body := `{"userID":""}`
	req := httptest.NewRequest(http.MethodPost, "/movies/test1/seats/A1/hold", strings.NewReader(body))
	req.SetPathValue("movieID", "test1")
	req.SetPathValue("seatID", "A1")
	w := httptest.NewRecorder()

	h.HoldSeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHoldSeat_InvalidJSON(t *testing.T) {
	svc := booking.NewService(&mockStore{})
	h := booking.NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/movies/test1/seats/A1/hold", strings.NewReader("{bad"))
	req.SetPathValue("movieID", "test1")
	req.SetPathValue("seatID", "A1")
	w := httptest.NewRecorder()

	h.HoldSeat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestConfirmSession_NotFound(t *testing.T) {
	store := &mockStore{
		confirmFn: func(_ context.Context, _, _ string) (booking.Booking, error) {
			return booking.Booking{}, booking.ErrSessionNotFound
		},
	}
	svc := booking.NewService(store)
	h := booking.NewHandler(svc)

	body := `{"userID":"user-1"}`
	req := httptest.NewRequest(http.MethodPut, "/sessions/nonexistent/confirm", strings.NewReader(body))
	req.SetPathValue("sessionID", "nonexistent")
	w := httptest.NewRecorder()

	h.ConfirmSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestReleaseSession_NotFound(t *testing.T) {
	store := &mockStore{
		releaseFn: func(_ context.Context, _, _ string) error {
			return booking.ErrSessionNotFound
		},
	}
	svc := booking.NewService(store)
	h := booking.NewHandler(svc)

	body := `{"userID":"user-1"}`
	req := httptest.NewRequest(http.MethodDelete, "/sessions/nonexistent", strings.NewReader(body))
	req.SetPathValue("sessionID", "nonexistent")
	w := httptest.NewRecorder()

	h.ReleaseSession(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListSeats_Success(t *testing.T) {
	store := &mockStore{
		listFn: func(_ context.Context, _ string) ([]booking.Booking, error) {
			return []booking.Booking{
				{SeatID: "A1", UserID: "user-1", Status: "held"},
				{SeatID: "B2", UserID: "user-2", Status: "confirmed"},
			}, nil
		},
	}
	svc := booking.NewService(store)
	h := booking.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/movies/test1/seats", nil)
	req.SetPathValue("movieID", "test1")
	w := httptest.NewRecorder()

	h.ListSeats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var seats []map[string]any
	json.NewDecoder(w.Body).Decode(&seats)
	if len(seats) != 2 {
		t.Errorf("expected 2 seats, got %d", len(seats))
	}
}
