package booking

import (
	"context"
	"log"
	"time"
)

// StartHoldExpiry runs a background goroutine that cleans up expired holds.
// It stops when the context is cancelled.
func StartHoldExpiry(ctx context.Context, store *PostgresStore, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("hold expiry worker stopped")
			return
		case <-ticker.C:
			cleaned, err := store.CleanExpiredHolds(ctx)
			if err != nil {
				log.Printf("hold expiry error: %v", err)
				continue
			}
			if cleaned > 0 {
				log.Printf("cleaned %d expired holds", cleaned)
			}
		}
	}
}
