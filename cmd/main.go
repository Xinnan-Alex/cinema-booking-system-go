package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Xinnan-Alex/cinema/internal/adapters/postgres"
	"github.com/Xinnan-Alex/cinema/internal/booking"
	"github.com/Xinnan-Alex/cinema/internal/config"
	"github.com/Xinnan-Alex/cinema/internal/health"
	"github.com/Xinnan-Alex/cinema/internal/middleware"
	"github.com/Xinnan-Alex/cinema/internal/movie"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool := postgres.NewPool(ctx, cfg.DatabaseURL)
	defer pool.Close()

	if err := postgres.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	pgStore := booking.NewPostgresStore(pool, cfg.HoldTTL)
	go booking.StartHoldExpiry(ctx, pgStore, 30*time.Second)

	bookingSvc := booking.NewService(pgStore)
	bookingHandler := booking.NewHandler(bookingSvc)

	movieStore := movie.NewPostgresStore(pool)
	movieSvc := movie.NewService(movieStore)
	movieHandler := movie.NewHandler(movieSvc)

	healthHandler := health.NewHandler(pool)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthHandler.Healthz)
	mux.Handle("GET /", http.FileServer(http.Dir("static")))
	mux.HandleFunc("GET /movies", movieHandler.ListMovies)

	mux.HandleFunc("GET /movies/{movieID}/seats", bookingHandler.ListSeats)
	mux.HandleFunc("POST /movies/{movieID}/seats/{seatID}/hold", bookingHandler.HoldSeat)
	mux.HandleFunc("PUT /sessions/{sessionID}/confirm", bookingHandler.ConfirmSession)
	mux.HandleFunc("DELETE /sessions/{sessionID}", bookingHandler.ReleaseSession)

	var handler http.Handler = mux
	handler = middleware.Logger(handler)
	handler = middleware.Recover(handler)
	handler = middleware.CORS(handler)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: handler,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Println("server stopped")
}
