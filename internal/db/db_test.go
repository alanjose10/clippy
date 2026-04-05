package db_test

import (
	"bytes"
	"testing"
	"time"

	"clippy/internal/db"
	"clippy/internal/model"
)

// newTestDB opens an in-memory SQLite database for testing.
// It is closed automatically when the test finishes.
func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// --- Rooms ---

func TestCreateRoom_And_GetRoom(t *testing.T) {
	database := newTestDB(t)

	room := &model.Room{ID: "work", CreatedAt: time.Now()}
	if err := database.CreateRoom(room); err != nil {
		t.Fatal(err)
	}

	got, err := database.GetRoom("work")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "work" {
		t.Errorf("expected ID 'work', got %q", got.ID)
	}
}

func TestGetRoom_NotFound(t *testing.T) {
	database := newTestDB(t)

	_, err := database.GetRoom("nonexistent")
	if err != db.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListRooms(t *testing.T) {
	database := newTestDB(t)

	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})
	database.CreateRoom(&model.Room{ID: "home", CreatedAt: time.Now()})

	rooms, err := database.ListRooms()
	if err != nil {
		t.Fatal(err)
	}
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}
}

func TestDeleteRoom(t *testing.T) {
	database := newTestDB(t)

	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})

	if err := database.DeleteRoom("work"); err != nil {
		t.Fatal(err)
	}

	_, err := database.GetRoom("work")
	if err != db.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDeleteRoom_CascadesSnippets(t *testing.T) {
	database := newTestDB(t)

	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})
	database.CreateSnippet(&model.DBSnippet{
		ID:         "s1",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	database.DeleteRoom("work")

	// Snippet should be gone too (ON DELETE CASCADE)
	_, err := database.GetSnippet("s1")
	if err != db.ErrNotFound {
		t.Errorf("expected snippet to be deleted with room, got %v", err)
	}
}

// --- Snippets ---

func TestCreateSnippet_And_GetSnippet(t *testing.T) {
	database := newTestDB(t)
	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})

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

	if err := database.CreateSnippet(s); err != nil {
		t.Fatal(err)
	}

	got, err := database.GetSnippet("abc123")
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
	database := newTestDB(t)

	_, err := database.GetSnippet("nonexistent")
	if err != db.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListSnippets_ReturnsOnlyNonExpired(t *testing.T) {
	database := newTestDB(t)
	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})

	// Live snippet
	database.CreateSnippet(&model.DBSnippet{
		ID:         "live",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	// Expired snippet
	database.CreateSnippet(&model.DBSnippet{
		ID:         "expired",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	snippets, err := database.ListSnippets("work")
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
	database := newTestDB(t)
	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})

	now := time.Now()
	database.CreateSnippet(&model.DBSnippet{
		ID:         "s1",
		RoomID:     "work",
		Name:       "original",
		ContentEnc: []byte("old"),
		Language:   "plaintext",
		ExpiresAt:  now.Add(time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	got, _ := database.GetSnippet("s1")
	got.Name = "updated"
	got.ContentEnc = []byte("new")
	got.Language = "go"
	got.UpdatedAt = time.Now()

	if err := database.UpdateSnippet(got); err != nil {
		t.Fatal(err)
	}

	refreshed, _ := database.GetSnippet("s1")
	if refreshed.Name != "updated" {
		t.Errorf("Name: got %q", refreshed.Name)
	}
	if refreshed.Language != "go" {
		t.Errorf("Language: got %q", refreshed.Language)
	}
}

func TestDeleteExpired(t *testing.T) {
	database := newTestDB(t)
	database.CreateRoom(&model.Room{ID: "work", CreatedAt: time.Now()})

	database.CreateSnippet(&model.DBSnippet{
		ID:         "expired1",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-2 * time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	database.CreateSnippet(&model.DBSnippet{
		ID:         "expired2",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(-time.Minute),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	database.CreateSnippet(&model.DBSnippet{
		ID:         "live",
		RoomID:     "work",
		ContentEnc: []byte("blob"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})

	n, err := database.DeleteExpired()
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}

	// Live snippet should still be there
	_, err = database.GetSnippet("live")
	if err != nil {
		t.Errorf("live snippet should not be deleted: %v", err)
	}
}
