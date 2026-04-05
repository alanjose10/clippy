package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"clippy/internal/model"
)

func (d *DB) CreateSnippet(ctx context.Context, snippet *model.DBSnippet) error {
	if snippet == nil {
		return fmt.Errorf("snippet cannot be nil")
	}

	if snippet.ID == "" {
		return fmt.Errorf("snippet id cannot be empty")
	}

	if snippet.CreatedAt.IsZero() {
		return fmt.Errorf("created at cannot be empty")
	}

	if snippet.ExpiresAt.IsZero() {
		return fmt.Errorf("expires at cannot be empty")
	}

	query := `
		INSERT INTO snippets (id, room_id, name, content_enc, language, tags, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := d.db.ExecContext(
		ctx,
		query,
		snippet.ID,
		snippet.RoomID,
		snippet.Name,
		snippet.ContentEnc,
		snippet.Language,
		snippet.Tags,
		snippet.ExpiresAt.Unix(),
		snippet.CreatedAt.Unix(),
		snippet.UpdatedAt.Unix(),
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) GetSnippet(ctx context.Context, id string) (*model.DBSnippet, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}
	var snippet model.DBSnippet
	var expiresAt int64
	var createdAt int64
	var updatedAt int64
	var name sql.NullString

	query := `SELECT id, room_id, name, content_enc, language, tags, expires_at, created_at, updated_at
	FROM snippets WHERE id = ?`
	res := d.db.QueryRowContext(ctx, query, id)
	err := res.Scan(
		&snippet.ID,
		&snippet.RoomID,
		&name,
		&snippet.ContentEnc,
		&snippet.Language,
		&snippet.Tags,
		&expiresAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {

		// if the room with id does not exist
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("querying snippet %q: %w", id, ErrNotFound)
		}

		return nil, err
	}

	if name.Valid {
		snippet.Name = name.String
	}

	snippet.ExpiresAt = time.Unix(expiresAt, 0)
	snippet.CreatedAt = time.Unix(createdAt, 0)
	snippet.UpdatedAt = time.Unix(updatedAt, 0)

	return &snippet, nil
}

func (d *DB) ListSnippets(ctx context.Context, roomID string) ([]*model.DBSnippet, error) {
	// ordering by DESC to show newest snippet first
	query := `
		SELECT id, room_id, name, content_enc, language, tags, expires_at, created_at, updated_at FROM snippets
		WHERE room_id = ? AND expires_at > unixepoch('now')
		ORDER BY created_at DESC
	`
	r, err := d.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var snippets []*model.DBSnippet
	for r.Next() {
		var snippet model.DBSnippet
		var expiresAt int64
		var createdAt int64
		var updatedAt int64
		var name sql.NullString

		if err := r.Scan(
			&snippet.ID,
			&snippet.RoomID,
			&name,
			&snippet.ContentEnc,
			&snippet.Language,
			&snippet.Tags,
			&expiresAt,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		snippet.ExpiresAt = time.Unix(expiresAt, 0)
		snippet.CreatedAt = time.Unix(createdAt, 0)
		snippet.UpdatedAt = time.Unix(updatedAt, 0)

		if name.Valid {
			snippet.Name = name.String
		}

		snippets = append(snippets, &snippet)
	}

	if err := r.Err(); err != nil {
		return nil, err
	}

	return snippets, nil
}

func (d *DB) UpdateSnippet(ctx context.Context, snippet *model.DBSnippet) error {
	if snippet == nil {
		return fmt.Errorf("snippet cannot be nil")
	}

	if snippet.ID == "" {
		return fmt.Errorf("snippet id cannot be empty")
	}

	query := `UPDATE snippets SET
	name = ?,
	content_enc = ?,
	language = ?,
	tags = ?,
	expires_at = ?,
	updated_at = ?
	WHERE id = ?`

	_, err := d.db.ExecContext(
		ctx,
		query,
		snippet.Name,
		snippet.ContentEnc,
		snippet.Language,
		snippet.Tags,
		snippet.ExpiresAt.Unix(),
		snippet.UpdatedAt.Unix(),
		snippet.ID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (d *DB) DeleteExpired(ctx context.Context) (int, error) {
	query := `DELETE from snippets where expires_at <= unixepoch('now')`
	res, err := d.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
