package booking

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultHoldTTL = 2 * time.Minute

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore {
	return &RedisStore{
		rdb: rdb,
	}
}

func sessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}

func (s *RedisStore) Book(b Booking) (Booking, error) {
	session, err := s.hold(b)
	if err != nil {
		return Booking{}, err
	}
	log.Printf("Session booked %v", session)
	return session, nil
}

func (s *RedisStore) ListBookings(movieID string) []Booking {
	pattern := fmt.Sprintf("seat:%s:*", movieID)
	bookings := []Booking{}

	ctx := context.Background()
	itr := s.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for itr.Next(ctx) {
		val, err := s.rdb.Get(ctx, itr.Val()).Result()
		if err != nil {
			continue
		}
		booking, err := parseBooking(val)
		if err != nil {
			continue
		}
		bookings = append(bookings, booking)
	}

	return bookings
}

func (s *RedisStore) Release(ctx context.Context, sessionID string, userID string) error {
	_, sk, err := s.getSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}

	s.rdb.Del(ctx, sk, sessionKey(sessionID))
	return nil
}

func (s *RedisStore) Confirm(ctx context.Context, sessionID string, userID string) (Booking, error) {
	session, sk, err := s.getSession(ctx, sessionID, userID)
	if err != nil {
		return Booking{}, nil
	}

	s.rdb.Persist(ctx, sk)
	s.rdb.Persist(ctx, sessionKey(sessionID))

	session.Status = "confirmed"
	data := Booking{
		ID:      session.ID,
		MovieID: session.MovieID,
		SeatID:  session.SeatID,
		Status:  session.Status,
	}
	val, _ := json.Marshal(data)
	s.rdb.Set(ctx, sk, val, 0)

	return session, nil
}

func (s *RedisStore) getSession(ctx context.Context, sessionID string, userID string) (Booking, string, error) {
	sk, err := s.rdb.Get(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		return Booking{}, "", err
	}

	booking, err := s.rdb.Get(ctx, sk).Result()
	if err != nil {
		return Booking{}, "", err
	}

	parsedBooking, err := parseBooking(booking)
	if err != nil {
		return Booking{}, "", err
	}

	return parsedBooking, sk, nil
}

func parseBooking(val string) (Booking, error) {
	var data Booking
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return Booking{}, err
	}
	return Booking{
		ID:      data.ID,
		MovieID: data.MovieID,
		SeatID:  data.SeatID,
		UserID:  data.UserID,
		Status:  data.Status,
	}, nil
}

func (s *RedisStore) hold(b Booking) (Booking, error) {
	id := uuid.New().String()
	now := time.Now()
	key := fmt.Sprintf("seat:%v:%v", b.MovieID, b.SeatID)
	b.ID = id
	ctx := context.Background()
	val, _ := json.Marshal(b)

	res := s.rdb.SetArgs(ctx, key, val, redis.SetArgs{
		Mode: string(redis.NX),
		TTL:  defaultHoldTTL,
	})
	ok := res.Val() == "OK"
	if !ok {
		return Booking{}, ErrSeatTaken
	}

	s.rdb.Set(ctx, sessionKey(id), key, defaultHoldTTL)

	return Booking{
		ID:        id,
		MovieID:   b.MovieID,
		SeatID:    b.SeatID,
		UserID:    b.UserID,
		Status:    "held",
		ExpiresAt: now.Add(defaultHoldTTL),
	}, nil
}
