package repo

import (
	"context"
	"time"
)

type HistoryRepo struct{ store *Store }

func (r *HistoryRepo) Create(ctx context.Context, h History) (History, error) {
	if h.Timestamp == "" {
		h.Timestamp = r.store.Clock().UTC().Format(time.RFC3339)
	}
	res, err := r.store.DB.ExecContext(ctx, "INSERT INTO history(timestamp, action, path, old_data, new_data) VALUES (?, ?, ?, ?, ?)",
		h.Timestamp, h.Action, h.Path, h.OldData, h.NewData)
	if err != nil {
		return History{}, err
	}
	id, _ := res.LastInsertId()
	h.ID = id
	return h, nil
}

func (r *HistoryRepo) List(ctx context.Context) ([]History, error) {
	rows, err := r.store.DB.QueryContext(ctx, "SELECT id, timestamp, action, path, old_data, new_data FROM history ORDER BY timestamp, id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []History
	for rows.Next() {
		var h History
		if err := rows.Scan(&h.ID, &h.Timestamp, &h.Action, &h.Path, &h.OldData, &h.NewData); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *HistoryRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.store.DB.ExecContext(ctx, "DELETE FROM history WHERE id=?", id)
	return err
}
