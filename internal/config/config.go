package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
)

type CurrencyChecker interface {
	CurrencyExists(ctx context.Context, code string) bool
}

type Options struct {
	CWD            string
	Home           string
	DetectCurrency func() (string, bool)
	CurrencyExists func(context.Context, string) bool
}

type Config struct {
	Currency string `json:"currency"`
}

type Loaded struct {
	Config  Config
	Path    string
	Warning string
	Created bool
}

func Load(ctx context.Context, opts Options) (Loaded, error) {
	if opts.CWD == "" {
		var err error
		opts.CWD, err = os.Getwd()
		if err != nil {
			return Loaded{}, err
		}
	}
	if opts.Home == "" {
		var err error
		opts.Home, err = os.UserHomeDir()
		if err != nil {
			return Loaded{}, err
		}
	}
	local := filepath.Join(opts.CWD, "config.jsonc")
	global := filepath.Join(opts.Home, ".config", "stuf", "config.jsonc")
	for _, path := range []string{local, global} {
		if _, err := os.Stat(path); err == nil {
			cfg, err := parseFile(path)
			if err != nil {
				return Loaded{}, err
			}
			if err := validate(ctx, &cfg, opts.CurrencyExists); err != nil {
				return Loaded{}, fmt.Errorf("%s: %w", path, err)
			}
			return Loaded{Config: cfg, Path: path}, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return Loaded{}, err
		}
	}
	code := "USD"
	warning := "app currency defaulted to USD; edit config.jsonc to change it"
	if opts.DetectCurrency != nil {
		if detected, ok := opts.DetectCurrency(); ok && strings.TrimSpace(detected) != "" {
			code = strings.ToUpper(strings.TrimSpace(detected))
			warning = ""
		}
	}
	cfg := Config{Currency: code}
	if err := validate(ctx, &cfg, opts.CurrencyExists); err != nil {
		return Loaded{}, err
	}
	if err := os.MkdirAll(filepath.Dir(global), 0o755); err != nil {
		return Loaded{}, err
	}
	content := fmt.Sprintf("{\n  // stuf config\n  // docs: README.md#config\n  \"currency\": %q,\n}\n", code)
	if err := os.WriteFile(global, []byte(content), 0o600); err != nil {
		return Loaded{}, err
	}
	return Loaded{Config: cfg, Path: global, Warning: warning, Created: true}, nil
}

func Parse(data []byte) (Config, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return Config{}, errors.New("config file is empty")
	}
	ast, err := hujson.Parse(data)
	if err != nil {
		return Config{}, fmt.Errorf("malformed JSONC: %w", err)
	}
	ast.Standardize()
	var cfg Config
	if err := json.Unmarshal(ast.Pack(), &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config shape: %w", err)
	}
	return cfg, nil
}

func parseFile(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(b)
}

func validate(ctx context.Context, cfg *Config, exists func(context.Context, string) bool) error {
	if strings.TrimSpace(cfg.Currency) == "" {
		return errors.New("currency is required")
	}
	cfg.Currency = strings.ToUpper(strings.TrimSpace(cfg.Currency))
	if exists != nil && !exists(ctx, cfg.Currency) {
		return fmt.Errorf("unknown currency %q", cfg.Currency)
	}
	return nil
}
