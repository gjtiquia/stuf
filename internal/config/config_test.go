package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseJSONC(t *testing.T) {
	for _, body := range []string{
		`{"currency":"HKD"}`,
		"{// hi\n\"currency\":\"HKD\"}",
		"{/* hi */\"currency\":\"HKD\"}",
		"{\"currency\":\"HKD\",}",
	} {
		cfg, err := Parse([]byte(body))
		if err != nil {
			t.Fatalf("Parse(%q): %v", body, err)
		}
		if cfg.Currency != "HKD" {
			t.Fatalf("currency = %q", cfg.Currency)
		}
	}
}

func TestRejectInvalidConfig(t *testing.T) {
	for _, body := range []string{"", "{", "[]", `{"currency":""}`} {
		cfg, err := Parse([]byte(body))
		if err == nil {
			err = validate(context.Background(), &cfg, func(context.Context, string) bool { return true })
		}
		if err == nil {
			t.Fatalf("expected invalid config for %q", body)
		}
	}
}

func TestLoadCreatesGlobalDefaultWithFallbackWarning(t *testing.T) {
	root := t.TempDir()
	loaded, err := Load(context.Background(), Options{
		CWD:  root,
		Home: root,
		DetectCurrency: func() (string, bool) {
			return "", false
		},
		CurrencyExists: func(_ context.Context, code string) bool { return code == "USD" },
	})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Config.Currency != "USD" || loaded.Warning == "" || !loaded.Created {
		t.Fatalf("unexpected load: %+v", loaded)
	}
	b, err := os.ReadFile(loaded.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "README.md#config") {
		t.Fatalf("generated config missing docs comment: %s", b)
	}
	if _, err := Parse(b); err != nil {
		t.Fatalf("generated config is invalid: %v", err)
	}
}

func TestLoadLocalPrecedenceAndUnknownCurrency(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.jsonc")
	if err := os.WriteFile(path, []byte(`{"currency":"NOPE"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(context.Background(), Options{
		CWD: root, Home: root,
		CurrencyExists: func(_ context.Context, code string) bool { return code == "USD" },
	})
	if err == nil || !strings.Contains(err.Error(), "unknown currency") {
		t.Fatalf("expected unknown currency error, got %v", err)
	}
}
