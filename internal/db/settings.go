package db

import (
	"context"
	"database/sql"
	"fmt"
)

// GetSetting retrieves a setting value by key.
// Returns empty string and no error if the key doesn't exist.
func (db *DB) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := db.conn.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting setting %q: %w", key, err)
	}
	return value, nil
}

// SetSetting creates or updates a setting.
func (db *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := db.conn.ExecContext(ctx,
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	if err != nil {
		return fmt.Errorf("setting %q: %w", key, err)
	}
	return nil
}

// DeleteSetting removes a setting by key.
func (db *DB) DeleteSetting(ctx context.Context, key string) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("deleting setting %q: %w", key, err)
	}
	return nil
}
