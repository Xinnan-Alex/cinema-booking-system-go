package movie

import "context"

type Movie struct {
	ID          string
	Title       string
	Rows        int
	SeatsPerRow int
}

type MovieStore interface {
	List(ctx context.Context) ([]Movie, error)
}
