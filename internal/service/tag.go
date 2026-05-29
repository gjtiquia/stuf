package service

import (
	"context"
	"errors"
	"strings"

	"stuf/internal/repo"
)

type TagService struct {
	Store   *repo.Store
	Tags    *repo.TagRepo
	History HistoryService
}

type tagMutationData struct {
	Tag repo.Tag
}

func ValidateTagName(name string) error {
	if name == "" {
		return errors.New("tag name is required")
	}
	for _, segment := range strings.Split(name, "/") {
		if !slugPattern.MatchString(segment) {
			return errors.New("tag name must be a strict slug; slash hierarchy is allowed")
		}
	}
	return nil
}

func (s TagService) Create(ctx context.Context, name, notes string) (repo.Tag, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateTagName(name); err != nil {
		return repo.Tag{}, SessionEntry{}, err
	}
	var out repo.Tag
	var entry SessionEntry
	err := s.Store.WithWriteTx(ctx, func() error {
		tag, err := s.Tags.Create(ctx, name, notes)
		if err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "create", "/tags/"+tag.Name, nil, tagMutationData{Tag: tag}, func(ctx context.Context) error {
			return s.Tags.DeleteIfUnused(ctx, tag.ID)
		})
		if err != nil {
			return err
		}
		out, entry = tag, e
		return nil
	})
	return out, entry, err
}

func (s TagService) Update(ctx context.Context, id int64, name, notes string) (repo.Tag, SessionEntry, error) {
	name = strings.TrimSpace(name)
	if err := ValidateTagName(name); err != nil {
		return repo.Tag{}, SessionEntry{}, err
	}
	old, err := s.Tags.GetByID(ctx, id)
	if err != nil {
		return repo.Tag{}, SessionEntry{}, err
	}
	next := old
	next.Name, next.Notes = name, notes
	var out repo.Tag
	var entry SessionEntry
	err = s.Store.WithWriteTx(ctx, func() error {
		updated, err := s.Tags.Update(ctx, next)
		if err != nil {
			return err
		}
		e, err := s.History.Record(ctx, "edit", "/tags/"+updated.Name, tagMutationData{Tag: old}, tagMutationData{Tag: updated}, func(ctx context.Context) error {
			_, err := s.Tags.Update(ctx, old)
			return err
		})
		if err != nil {
			return err
		}
		out, entry = updated, e
		return nil
	})
	return out, entry, err
}

func (s TagService) List(ctx context.Context) ([]repo.Tag, error) {
	return s.Tags.List(ctx)
}

func (s TagService) GetByName(ctx context.Context, name string) (repo.Tag, error) {
	return s.Tags.GetByName(ctx, name)
}
