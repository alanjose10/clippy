package worker

import (
	"context"
	"log"
	"time"
)

type expirer interface {
	// Delete the expired resources
	DeleteExpired(context.Context) (int, error)
}

func Start(ctx context.Context, db expirer, d time.Duration) {
	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := db.DeleteExpired(ctx)
				if err != nil {
					log.Printf("expiry worker: %v", err)
				}
				if n > 0 {
					log.Printf("expiry worker: deleted %d snippets", n)
				}
			}
		}
	}()
}
