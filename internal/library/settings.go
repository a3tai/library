package library

import (
	"context"
	"time"
)

// GetSettings returns every key in the settings table as a flat map. Empty
// values are preserved so the caller can tell "explicitly cleared" apart
// from "never set" (the latter just won't be in the map).
func (s *Store) GetSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// SetSettings upserts every key/value pair in `kv` atomically. Keys with
// empty values are kept (callers use `""` to mean "fall back to env /
// default" — explicit clearing is distinct from "never set").
func (s *Store) SetSettings(ctx context.Context, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO settings(key, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	now := time.Now().UTC()
	for k, v := range kv {
		if _, err := stmt.ExecContext(ctx, k, v, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
