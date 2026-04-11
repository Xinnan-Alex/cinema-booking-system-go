package main

import (
	"log"
	"net/http"

	"github.com/Xinnan-Alex/cinema/internal/adapters/redis"
	"github.com/Xinnan-Alex/cinema/internal/booking"
	"github.com/Xinnan-Alex/cinema/internal/utils"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /movies", listMovies)

	mux.Handle("GET /", http.FileServer(http.Dir("static")))

	redisStore := booking.NewRedisStore(redis.NewClient("localhost:6379"))
	svc := booking.NewService(redisStore)

	bookingHandler := booking.NewHandler(svc)

	mux.HandleFunc("GET /movies/{movieID}/seats", bookingHandler.ListSeats)
	mux.HandleFunc("POST /movies/{movieID}/seats/{seatID}/hold", bookingHandler.HoldSeat)

	mux.HandleFunc("PUT /sessions/{sessionID}/confirm", bookingHandler.ConfirmSession)
	mux.HandleFunc("DELETE /sessions/{sessionID}", bookingHandler.ReleaseSession)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

type movieResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seatsPerRow"`
}

var movies = []movieResponse{
	{ID: "test1", Title: "test1", Rows: 5, SeatsPerRow: 8},
	{ID: "test2", Title: "test2", Rows: 4, SeatsPerRow: 6},
}

func listMovies(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, movies)
}
