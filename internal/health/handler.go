package health

import (
	"net/http"

	"github.com/Xinnan-Alex/cinema/internal/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *handler {
	return &handler{pool: pool}
}

func (h *handler) Healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
