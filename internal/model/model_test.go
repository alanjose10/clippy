package model_test

import (
	"fmt"
	"testing"
	"time"

	"clippy/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GenerateID ---

func TestGenerateID(t *testing.T) {
	id := model.GenerateID()
	assert.NotZero(t, id)
	assert.Len(t, id, 6)
}

func TestGenerateID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for range 200 {
		id := model.GenerateID()
		if seen[id] {
			t.Fatalf("duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}

// --- ParseTags ---

func TestParseTags(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"db,urgent", []string{"db", "urgent"}},
		{"db, urgent", []string{"db", "urgent"}},    // spaces trimmed
		{" db , urgent ", []string{"db", "urgent"}}, // leading/trailing spaces
		{"single", []string{"single"}},
		{"", []string{}},
		{"   ", []string{}},
		{"db,,urgent", []string{"db", "urgent"}}, // empty segment ignored
	}
	for _, c := range cases {
		got := model.ParseTags(c.input)
		if len(got) != len(c.want) {
			t.Errorf("ParseTags(%q) = %v, want %v", c.input, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("ParseTags(%q)[%d] = %q, want %q", c.input, i, got[i], c.want[i])
			}
		}
	}
}

// --- JoinTags ---

func TestJoinTags(t *testing.T) {
	cases := []struct {
		input []string
		want  string
	}{
		{[]string{"db", "urgent"}, "db,urgent"},
		{[]string{"single"}, "single"},
		{[]string{}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		got := model.JoinTags(c.input)
		if got != c.want {
			t.Errorf("JoinTags(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}

// --- MatchesAllTags ---

func TestMatchesAllTags(t *testing.T) {
	snippet := []string{"db", "urgent", "migration"}

	cases := []struct {
		filter []string
		want   bool
	}{
		{[]string{"db"}, true},
		{[]string{"db", "urgent"}, true},
		{[]string{"db", "urgent", "migration"}, true},
		{[]string{"db", "missing"}, false},
		{[]string{"missing"}, false},
		{[]string{}, true}, // empty filter matches everything
		{nil, true},        // nil filter matches everything
	}
	for _, c := range cases {
		got := model.MatchesAllTags(snippet, c.filter)
		if got != c.want {
			t.Errorf("MatchesAllTags(%v, %v) = %v, want %v", snippet, c.filter, got, c.want)
		}
	}
}

// --- UniqueTags ---

func TestUniqueTags_Empty(t *testing.T) {
	tags := model.UniqueTags([]model.Snippet{})
	if len(tags) != 0 {
		t.Errorf("expected empty, got %v", tags)
	}
}

func TestUniqueTags_Deduplicates(t *testing.T) {
	snippets := []model.Snippet{
		{Tags: []string{"db", "urgent"}},
		{Tags: []string{"db", "api"}},
		{Tags: []string{"frontend"}},
	}
	tags := model.UniqueTags(snippets)

	want := map[string]bool{"db": true, "urgent": true, "api": true, "frontend": true}
	if len(tags) != len(want) {
		t.Errorf("expected %d unique tags, got %d: %v", len(want), len(tags), tags)
	}
	for _, tag := range tags {
		if !want[tag] {
			t.Errorf("unexpected tag %q in result", tag)
		}
	}
}

func TestUniqueTags_Sorted(t *testing.T) {
	snippets := []model.Snippet{
		{Tags: []string{"zzz", "aaa", "mmm"}},
	}
	tags := model.UniqueTags(snippets)
	for i := 1; i < len(tags); i++ {
		if tags[i] < tags[i-1] {
			t.Errorf("tags not sorted: %v", tags)
		}
	}
}

// --- DBSnippet.ToSnippet ---

func TestDBSnippet_ToSnippet(t *testing.T) {
	now := time.Now()
	ds := &model.DBSnippet{
		ID:         "abc123",
		RoomID:     "work",
		Name:       "my-snippet",
		ContentEnc: []byte("plaintext"), // fake — decrypt func just returns it
		Language:   "go",
		Tags:       "db,urgent",
		ExpiresAt:  now.Add(24 * time.Hour),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Use a no-op decrypt for testing
	decrypt := func(b []byte) ([]byte, error) { return b, nil }

	s, err := ds.ToSnippet(decrypt)
	if err != nil {
		t.Fatal(err)
	}
	require.NotNil(t, s)

	if s.ID != "abc123" {
		t.Errorf("ID: got %q", s.ID)
	}
	if s.Content != "plaintext" {
		t.Errorf("Content: got %q", s.Content)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "db" || s.Tags[1] != "urgent" {
		t.Errorf("Tags: got %v", s.Tags)
	}
}

func TestDBSnippet_ToSnippet_DecryptError(t *testing.T) {
	ds := &model.DBSnippet{
		ID:         "abc123",
		RoomID:     "work",
		ContentEnc: []byte("bad-data"),
		Language:   "plaintext",
		ExpiresAt:  time.Now().Add(time.Hour),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	decrypt := func(b []byte) ([]byte, error) {
		return nil, fmt.Errorf("decryption failed")
	}

	_, err := ds.ToSnippet(decrypt)
	if err == nil {
		t.Fatal("expected error when decrypt fails")
	}
}
