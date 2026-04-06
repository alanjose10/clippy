package worker_test

import (
	"context"
	"testing"
	"time"

	"clippy/internal/db"
	"clippy/internal/model"
	"clippy/internal/worker"
)

func TestExpiryWorker_DeletesExpiredSnippets(t *testing.T) {
	ctx := context.Background()

	database, err := db.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "s1",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-1 * time.Minute),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Run with a very short interval so the worker fires quickly in tests
	worker.Start(workerCtx, database, 10*time.Millisecond)

	time.Sleep(50 * time.Millisecond)

	// After the worker has run, there should be no expired rows left
	n, err := database.DeleteExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected 0 rows remaining after worker ran, but DeleteExpired removed %d more", n)
	}
}
