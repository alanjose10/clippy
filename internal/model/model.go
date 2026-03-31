package model

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"
)

type DecryptFn func(b []byte) ([]byte, error)

type Room struct {
	ID        string
	PinHash   string
	CreatedAt time.Time
}

type Snippet struct {
	ID        string
	RoomID    string
	Name      string
	Language  string
	Tags      []string
	Content   string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DBSnippet struct {
	ID         string
	RoomID     string
	Name       string
	ContentEnc []byte
	Language   string
	Tags       string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (s *DBSnippet) ToSnippet(decrypt DecryptFn) (*Snippet, error) {
	tags := ParseTags(s.Tags)

	content, err := decrypt(s.ContentEnc)
	if err != nil {
		return nil, fmt.Errorf("decrypting content: %w", err)
	}

	return &Snippet{
		ID:        s.ID,
		Name:      s.Name,
		Tags:      tags,
		Content:   string(content),
		RoomID:    s.RoomID,
		Language:  s.Language,
		ExpiresAt: s.ExpiresAt,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}, nil
}

const alphaNums = "abcdefghijklmnopqrstuvwxyz0123456789"

// GenerateID generates a random UUID string
func GenerateID() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = alphaNums[rand.IntN(len(alphaNums))]
	}
	return string(b)
}

// ParseTags splits the tags separated by , and returns them as a slice of string
func ParseTags(tagsStr string) []string {
	tags := []string{}

	// Remove spaces
	tagsStr = strings.ReplaceAll(tagsStr, " ", "")

	if tagsStr != "" {
		for tag := range strings.SplitSeq(tagsStr, ",") {
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	return tags
}

// JoinTags joins the slice of tags with , and returns a string
func JoinTags(tags []string) string {
	return strings.Join(tags, ",")
}

// MatchesAllTags returns true of all the filters are present in the provided snippet tags slice
func MatchesAllTags(snippetTags []string, filter []string) bool {
	snippetTagsMap := make(map[string]struct{})
	for _, t := range snippetTags {
		snippetTagsMap[t] = struct{}{}
	}
	for _, f := range filter {
		if _, ok := snippetTagsMap[f]; !ok {
			return false
		}
	}
	return true
}

// UniqueTags returns the distinct slice of the provided tags slice
func UniqueTags(snippets []Snippet) []string {
	tagsMap := make(map[string]struct{})
	for _, s := range snippets {
		for _, t := range s.Tags {
			tagsMap[t] = struct{}{}
		}
	}

	keys := make([]string, 0, len(tagsMap))
	for k := range tagsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
