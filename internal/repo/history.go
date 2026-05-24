package repo

import (
	"context"
	"time"

	"stuf/internal/db"
)

type HistoryRepo struct{ store *Store }

func (r *HistoryRepo) Create(ctx context.Context, h History) (History, error) {
	if h.Timestamp == "" {
		h.Timestamp = r.store.Clock().UTC().Format(time.RFC3339)
	}
	res, err := r.store.Q.CreateHistory(ctx, db.CreateHistoryParams{
		Timestamp: h.Timestamp,
		Action:    h.Action,
		Path:      h.Path,
		OldData:   nullString(h.OldData),
		NewData:   nullString(h.NewData),
	})
	if err != nil {
		return History{}, err
	}
	id, _ := res.LastInsertId()
	h.ID = id
	return h, nil
}

func (r *HistoryRepo) List(ctx context.Context) ([]History, error) {
	rows, err := r.store.Q.ListHistory(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]History, len(rows))
	for i, row := range rows {
		out[i] = historyFromDB(row)
	}
	return out, nil
}

func (r *HistoryRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteHistory(ctx, id)
}
