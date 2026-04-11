package movie

import "context"

type Service struct {
	store MovieStore
}

func NewService(store MovieStore) *Service {
	return &Service{store: store}
}

func (s *Service) ListMovies(ctx context.Context) ([]Movie, error) {
	return s.store.List(ctx)
}
