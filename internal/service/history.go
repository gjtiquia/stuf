package service

import (
	"context"
	"encoding/json"
	"time"

	"stuf/internal/repo"
)

type UndoFunc func(context.Context) error

type SessionEntry struct {
	ID        int64
	Timestamp string
	Action    string
	Path      string
	Undo      UndoFunc
}

func (e SessionEntry) Line() string {
	ts, err := time.Parse(time.RFC3339, e.Timestamp)
	if err == nil {
		return ts.Local().Format("2006-01-02 15:04") + " " + e.Action + " " + e.Path
	}
	return e.Timestamp + " " + e.Action + " " + e.Path
}

type HistoryService struct {
	Repo *repo.HistoryRepo
	Now  func() time.Time
}

func (s HistoryService) Record(ctx context.Context, action, path string, oldData, newData any, undo UndoFunc) (SessionEntry, error) {
	oldJSON, err := optionalJSON(oldData)
	if err != nil {
		return SessionEntry{}, err
	}
	newJSON, err := optionalJSON(newData)
	if err != nil {
		return SessionEntry{}, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	h, err := s.Repo.Create(ctx, repo.History{
		Timestamp: now().UTC().Format(time.RFC3339),
		Action:    action,
		Path:      path,
		OldData:   oldJSON,
		NewData:   newJSON,
	})
	if err != nil {
		return SessionEntry{}, err
	}
	return SessionEntry{ID: h.ID, Timestamp: h.Timestamp, Action: action, Path: path, Undo: undo}, nil
}

func (s HistoryService) Undo(ctx context.Context, entry SessionEntry) error {
	if err := entry.Undo(ctx); err != nil {
		return err
	}
	return s.Repo.Delete(ctx, entry.ID)
}

func optionalJSON(v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}
