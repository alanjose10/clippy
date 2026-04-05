package db_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"clippy/internal/db"
	"clippy/internal/model"

	"github.com/stretchr/testify/require"
)

// newTestDB opens an in-memory SQLite database for testing.
// It is closed automatically when the test finishes.
func newTestDB(ctx context.Context, t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(ctx, ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// --- Rooms ---

func TestCreateRoom_And_GetRoom(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	room := &model.Room{ID: "work", CreatedAt: time.Now()}
	if err := database.CreateRoom(ctx, room); err != nil {
		t.Fatal(err)
	}

	got, err := database.GetRoom(ctx, "work")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "work" {
		t.Errorf("expected ID 'work', got %q", got.ID)
	}
}

func TestGetRoom_NotFound(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	_, err := database.GetRoom(ctx, "nonexistent")
	require.ErrorIs(t, err, db.ErrNotFound)
}

func TestListRooms(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})
	database.CreateRoom(ctx, &model.Room{ID: "home", CreatedAt: time.Now()})

	rooms, err := database.ListRooms(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}
}

func TestDeleteRoom(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})

	if err := database.DeleteRoom(ctx, "work"); err != nil {
		t.Fatal(err)
	}

	_, err := database.GetRoom(ctx, "work")

	require.ErrorIs(t, err, db.ErrNotFound)
}

func TestDeleteRoom_CascadesSnippets(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "s1",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	database.DeleteRoom(ctx, "work")

	// Snippet should be gone too (ON DELETE CASCADE)
	_, err := database.GetSnippet(ctx, "s1")

	require.ErrorIs(t, err, db.ErrNotFound)
}

// --- Snippets ---

func TestCreateSnippet_And_GetSnippet(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})

	now := time.Now()
	s := &model.DBSnippet{
		ID:         "abc123",
		RoomID:     "work",
		Name:       "my-snippet",
		ContentEnc: []byte("encrypted-blob"),
		Language:   "go",
		Tags:       "db,urgent",
		ExpiresAt:  now.Add(24 * time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := database.CreateSnippet(ctx, s); err != nil {
		t.Fatal(err)
	}

	got, err := database.GetSnippet(ctx, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "abc123" {
		t.Errorf("ID: got %q", got.ID)
	}
	if got.Name != "my-snippet" {
		t.Errorf("Name: got %q", got.Name)
	}
	if got.Language != "go" {
		t.Errorf("Language: got %q", got.Language)
	}
	if got.Tags != "db,urgent" {
		t.Errorf("Tags: got %q", got.Tags)
	}
	if !bytes.Equal(got.ContentEnc, []byte("encrypted-blob")) {
		t.Errorf("ContentEnc: got %v", got.ContentEnc)
	}
}

func TestGetSnippet_NotFound(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)

	_, err := database.GetSnippet(ctx, "nonexistent")

	require.ErrorIs(t, err, db.ErrNotFound)
}

func TestListSnippets_ReturnsOnlyNonExpired(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)
	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})

	// Live snippet
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "live",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	// Expired snippet
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "expired",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	snippets, err := database.ListSnippets(ctx, "work")
	if err != nil {
		t.Fatal(err)
	}
	if len(snippets) != 1 {
		t.Errorf("expected 1 live snippet, got %d", len(snippets))
	}
	if snippets[0].ID != "live" {
		t.Errorf("expected 'live', got %q", snippets[0].ID)
	}
}

func TestUpdateSnippet(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)
	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})

	now := time.Now()
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "s1",
		RoomID:     "work",
		Name:       "original",
		ContentEnc: []byte("old"),
		Language:   "plaintext",
		ExpiresAt:  now.Add(time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	got, _ := database.GetSnippet(ctx, "s1")
	got.Name = "updated"
	got.ContentEnc = []byte("new")
	got.Language = "go"
	got.UpdatedAt = time.Now()

	if err := database.UpdateSnippet(ctx, got); err != nil {
		t.Fatal(err)
	}

	refreshed, _ := database.GetSnippet(ctx, "s1")
	if refreshed.Name != "updated" {
		t.Errorf("Name: got %q", refreshed.Name)
	}
	if refreshed.Language != "go" {
		t.Errorf("Language: got %q", refreshed.Language)
	}
}

func TestDeleteExpired(t *testing.T) {
	ctx := context.Background()
	database := newTestDB(ctx, t)
	database.CreateRoom(ctx, &model.Room{ID: "work", CreatedAt: time.Now()})

	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "expired1",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-2 * time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "expired2",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-time.Minute),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	database.CreateSnippet(ctx, &model.DBSnippet{
		ID:         "live",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	n, err := database.DeleteExpired(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}

	// Live snippet should still be there
	_, err = database.GetSnippet(ctx, "live")
	if err != nil {
		t.Errorf("live snippet should not be deleted: %v", err)
	}
}
