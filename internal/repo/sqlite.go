package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pressly/goose/v3"

	"stuf/internal/db"
	"stuf/internal/migration"
	"stuf/internal/seed"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB    *sql.DB
	Q     *db.Queries
	Path  string
	mu    sync.Mutex
	Clock func() time.Time
	Acct  *AccountRepo
	Bal   *BalanceRepo
	Cur   *CurrencyRepo
	Hist  *HistoryRepo
}

func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	existed := true
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		existed = false
	} else if err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	store := &Store{DB: sqlDB, Q: db.New(sqlDB), Path: path, Clock: time.Now}
	store.Acct = &AccountRepo{store: store}
	store.Bal = &BalanceRepo{store: store}
	store.Cur = &CurrencyRepo{store: store}
	store.Hist = &HistoryRepo{store: store}
	if existed {
		var n int
		if err := sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&n); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("not a valid sqlite database: %w", err)
		}
		if n > 0 {
			app, err := store.Q.GetAppMetaApp(ctx)
			if err != nil || app != "stuf" {
				sqlDB.Close()
				return nil, fmt.Errorf("not a stuf database")
			}
		}
	}
	if err := store.migrate(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := store.verifyStuf(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := store.validateSchema(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err := store.SeedCurrencies(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) WithWriteLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn()
}

func (s *Store) Backup(ctx context.Context, now time.Time) (string, error) {
	var out string
	err := s.WithWriteLock(func() error {
		out = filepath.Join(filepath.Dir(s.Path), fmt.Sprintf("db.%s.sqlite", now.Format("2006-01-02-1504")))
		tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			return err
		}
		defer tx.Rollback()
		src, err := os.Open(s.Path)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			return err
		}
		return tx.Commit()
	})
	return out, err
}

func (s *Store) migrate(ctx context.Context) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	goose.SetBaseFS(migration.FS)
	if err := goose.UpContext(ctx, s.DB, "."); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (s *Store) verifyStuf(ctx context.Context) error {
	app, err := s.Q.GetAppMetaApp(ctx)
	if err != nil {
		return fmt.Errorf("not a stuf database: %w", err)
	}
	if app != "stuf" {
		return fmt.Errorf("not a stuf database: app=%q", app)
	}
	return nil
}

func (s *Store) validateSchema(ctx context.Context) error {
	for _, table := range []string{"app_meta", "currencies", "currency_rates", "accounts", "balances", "history"} {
		var name string
		err := s.DB.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			return fmt.Errorf("required schema missing table %s", table)
		}
	}
	return nil
}

type seedCurrency struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Scale           int    `json:"scale"`
	RateToUSDAmount int64  `json:"rate_to_usd_amount"`
	RateToUSDScale  int    `json:"rate_to_usd_scale"`
}

func (s *Store) SeedCurrencies(ctx context.Context) error {
	b, err := seed.FS.ReadFile("currencies.json")
	if err != nil {
		return err
	}
	var rows []seedCurrency
	if err := json.Unmarshal(b, &rows); err != nil {
		return err
	}
	now := s.Clock().UTC().Format(time.RFC3339)
	for _, row := range rows {
		if err := s.Q.UpsertCurrency(ctx, db.UpsertCurrencyParams{
			Code:      row.Code,
			Name:      row.Name,
			Scale:     int64(row.Scale),
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			return err
		}
		id, err := s.Q.GetCurrencyIDByCode(ctx, row.Code)
		if err != nil {
			return err
		}
		if err := s.Q.UpsertCurrencyRate(ctx, db.UpsertCurrencyRateParams{
			CurrencyID:      id,
			RateToUsdAmount: row.RateToUSDAmount,
			RateToUsdScale:  int64(row.RateToUSDScale),
			UpdatedAt:       now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SetCurrencyRate(ctx context.Context, code string, amount int64, scale int) error {
	now := s.Clock().UTC().Format(time.RFC3339)
	id, err := s.Q.GetCurrencyIDByCode(ctx, code)
	if err != nil {
		return err
	}
	return s.Q.UpsertCurrencyRate(ctx, db.UpsertCurrencyRateParams{
		CurrencyID:      id,
		RateToUsdAmount: amount,
		RateToUsdScale:  int64(scale),
		UpdatedAt:       now,
	})
}

func (s *Store) UpsertCurrencyNameOnly(ctx context.Context, code, name string) error {
	now := s.Clock().UTC().Format(time.RFC3339)
	return s.Q.UpsertCurrencyNameOnly(ctx, db.UpsertCurrencyNameOnlyParams{
		Code:      code,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	})
}
