package booking

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Xinnan-Alex/cinema/internal/utils"
)

type handler struct {
	svc *Service
}

func NewHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

type seatInfo struct {
	SeatID string `json:"seatID"`
	UserID string `json:"userID"`
	Booked bool   `json:"booked"`
}

func (h *handler) ListSeats(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieID")
	if movieID == "" {
		utils.WriteError(w, http.StatusBadRequest, "movieID is required")
		return
	}

	bookings, err := h.svc.ListBookings(r.Context(), movieID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to list seats")
		return
	}

	seats := make([]seatInfo, 0, len(bookings))
	for _, b := range bookings {
		seats = append(seats, seatInfo{
			SeatID: b.SeatID,
			UserID: b.UserID,
			Booked: true,
		})
	}

	utils.WriteJSON(w, http.StatusOK, seats)
}

type holdResponse struct {
	SessionID string `json:"sessionID"`
	MovieID   string `json:"movieID"`
	SeatID    string `json:"seatID"`
	ExpiresAt string `json:"expiresAt"`
}

type holdSeatRequest struct {
	UserID string `json:"userID"`
}

func (h *handler) HoldSeat(w http.ResponseWriter, r *http.Request) {
	movieID := r.PathValue("movieID")
	seatID := r.PathValue("seatID")

	var req holdSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		utils.WriteError(w, http.StatusBadRequest, "userID is required")
		return
	}

	session, err := h.svc.Book(r.Context(), Booking{
		UserID:  req.UserID,
		SeatID:  seatID,
		MovieID: movieID,
	})
	if err != nil {
		if errors.Is(err, ErrSeatTaken) {
			utils.WriteError(w, http.StatusConflict, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to hold seat")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, holdResponse{
		SeatID:    seatID,
		MovieID:   session.MovieID,
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	})
}

type sessionResponse struct {
	SessionID string `json:"sessionID"`
	MovieID   string `json:"movieID"`
	SeatID    string `json:"seatID"`
	UserID    string `json:"userID"`
	Status    string `json:"status"`
}

func (h *handler) ConfirmSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	var req holdSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		utils.WriteError(w, http.StatusBadRequest, "userID is required")
		return
	}

	session, err := h.svc.ConfirmSeat(r.Context(), sessionID, req.UserID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to confirm session")
		return
	}

	utils.WriteJSON(w, http.StatusOK, sessionResponse{
		SessionID: session.ID,
		MovieID:   session.MovieID,
		SeatID:    session.SeatID,
		UserID:    req.UserID,
		Status:    session.Status,
	})
}

func (h *handler) ReleaseSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	var req holdSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		utils.WriteError(w, http.StatusBadRequest, "userID is required")
		return
	}

	if err := h.svc.ReleaseSeat(r.Context(), sessionID, req.UserID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			utils.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "failed to release session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
