package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"clippy/internal/model"
)

// CreateRoom inserts a new room to the db
func (d *DB) CreateRoom(ctx context.Context, room *model.Room) error {
	if room.ID == "" {
		return fmt.Errorf("room id cannot be empty")
	}
	if room.CreatedAt.IsZero() {
		return fmt.Errorf("created at cannot be empty")
	}

	query := `
		INSERT INTO rooms (id, created_at, pin_hash)
		VALUES (?, ?, ?)
	`
	_, err := d.db.ExecContext(
		ctx,
		query,
		room.ID, room.CreatedAt.Unix(), room.PinHash,
	)
	if err != nil {
		return fmt.Errorf("creating room: %w", err)
	}
	return nil
}

// GetRoom returns a single room from the db
func (d *DB) GetRoom(ctx context.Context, id string) (*model.Room, error) {
	if id == "" {
		return nil, fmt.Errorf("room id cannot be empty")
	}
	var room model.Room
	var createdAt int64
	var pinHash sql.NullString
	query := `SELECT id, created_at, pin_hash from rooms WHERE id = ?`
	err := d.db.QueryRowContext(ctx, query, id).Scan(&room.ID, &createdAt, &pinHash)
	if err != nil {

		// if the room with id does not exist
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("querying room %q: %w", id, ErrNotFound)
		}

		return nil, err
	}

	room.CreatedAt = time.Unix(createdAt, 0)

	if pinHash.Valid {
		room.PinHash = pinHash.String
	}

	return &room, nil
}

// ListRooms returns a list of rooms
func (d *DB) ListRooms(ctx context.Context) ([]*model.Room, error) {
	query := `
		SELECT id, created_at, pin_hash FROM rooms
		ORDER BY created_at ASC
	`
	r, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var rooms []*model.Room
	for r.Next() {
		var room model.Room
		var createdAt int64
		var pinHash sql.NullString

		if err := r.Scan(&room.ID, &createdAt, &pinHash); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}

		room.CreatedAt = time.Unix(createdAt, 0)

		if pinHash.Valid {
			room.PinHash = pinHash.String
		}

		rooms = append(rooms, &room)
	}

	if err := r.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}

// DeleteRoom deletes a room from the db
func (d *DB) DeleteRoom(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("room id cannot be empty")
	}
	query := `DELETE from rooms where id = ?`
	_, err := d.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
