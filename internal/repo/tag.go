package repo

import (
	"context"
	"time"

	"stuf/internal/db"
)

type TagRepo struct{ store *Store }

func (r *TagRepo) Create(ctx context.Context, name, notes string) (Tag, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	res, err := r.store.Q.CreateTag(ctx, db.CreateTagParams{
		Name:      name,
		Notes:     notes,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return Tag{}, mapTagWriteErr(err, name)
	}
	id, _ := res.LastInsertId()
	return r.GetByID(ctx, id)
}

func (r *TagRepo) GetByID(ctx context.Context, id int64) (Tag, error) {
	row, err := r.store.Q.GetTagByID(ctx, id)
	if err != nil {
		return Tag{}, mapTagErr(err)
	}
	return tagFromDB(row), nil
}

func (r *TagRepo) GetByName(ctx context.Context, name string) (Tag, error) {
	row, err := r.store.Q.GetTagByName(ctx, name)
	if err != nil {
		return Tag{}, mapTagErr(err)
	}
	return tagFromDB(row), nil
}

func (r *TagRepo) List(ctx context.Context) ([]Tag, error) {
	rows, err := r.store.Q.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, row := range rows {
		out[i] = tagFromDB(row)
	}
	return out, nil
}

func (r *TagRepo) Update(ctx context.Context, t Tag) (Tag, error) {
	now := r.store.Clock().UTC().Format(time.RFC3339)
	if err := r.store.Q.UpdateTag(ctx, db.UpdateTagParams{
		Name:      t.Name,
		Notes:     t.Notes,
		UpdatedAt: now,
		ID:        t.ID,
	}); err != nil {
		return Tag{}, mapTagWriteErr(err, t.Name)
	}
	return r.GetByID(ctx, t.ID)
}

func (r *TagRepo) Delete(ctx context.Context, id int64) error {
	return r.store.Q.DeleteTag(ctx, id)
}

func (r *TagRepo) DeleteIfUnused(ctx context.Context, id int64) error {
	count, err := r.store.Q.CountAccountTagsByTagID(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return r.Delete(ctx, id)
}

func (r *TagRepo) ListByAccountID(ctx context.Context, accountID int64) ([]Tag, error) {
	rows, err := r.store.Q.ListTagsByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, row := range rows {
		out[i] = tagFromDB(row)
	}
	return out, nil
}

func (r *TagRepo) ListEffectiveByAccountID(ctx context.Context, accountID int64) ([]Tag, error) {
	rows, err := r.store.Q.ListEffectiveTagsByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]Tag, len(rows))
	for i, row := range rows {
		out[i] = tagFromDB(row)
	}
	return out, nil
}

func (r *TagRepo) SetAccountTags(ctx context.Context, accountID int64, tagIDs []int64) error {
	if err := r.store.Q.DeleteAccountTagsByAccountID(ctx, accountID); err != nil {
		return err
	}
	now := r.store.Clock().UTC().Format(time.RFC3339)
	seen := map[int64]bool{}
	for _, tagID := range tagIDs {
		if seen[tagID] {
			continue
		}
		seen[tagID] = true
		if err := r.store.Q.AddAccountTag(ctx, db.AddAccountTagParams{AccountID: accountID, TagID: tagID, CreatedAt: now}); err != nil {
			return err
		}
	}
	return nil
}
