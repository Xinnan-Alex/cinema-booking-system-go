package movie

import (
	"net/http"

	"github.com/Xinnan-Alex/cinema/internal/utils"
)

type handler struct {
	svc *Service
}

func NewHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

type movieResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seatsPerRow"`
}

func (h *handler) ListMovies(w http.ResponseWriter, r *http.Request) {
	movies, err := h.svc.ListMovies(r.Context())
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to list movies")
		return
	}

	resp := make([]movieResponse, 0, len(movies))
	for _, m := range movies {
		resp = append(resp, movieResponse{
			ID:          m.ID,
			Title:       m.Title,
			Rows:        m.Rows,
			SeatsPerRow: m.SeatsPerRow,
		})
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}
