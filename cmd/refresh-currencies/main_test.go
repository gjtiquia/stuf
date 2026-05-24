package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"stuf/internal/currencyseed"
)

func TestRunFromEnvUsesOverrideURLsAndOutput(t *testing.T) {
	dir := t.TempDir()
	ecbPath := filepath.Join(dir, "ecb.xml")
	metadataPath := filepath.Join(dir, "metadata.json")
	out := filepath.Join(dir, "currencies.json")
	if err := os.WriteFile(ecbPath, []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Envelope>
  <Cube>
    <Cube time="2026-05-22">
      <Cube currency="USD" rate="1.2"/>
      <Cube currency="JPY" rate="150"/>
    </Cube>
  </Cube>
</Envelope>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, []byte(`[
		  {"code":"USD","name":"US Dollar","decimals":2},
		  {"code":"EUR","name":"Euro","decimals":2},
		  {"code":"JPY","name":"Yen","decimals":0}
		]`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUF_ECB_RATES_URL", "file://"+ecbPath)
	t.Setenv("STUF_CURRENCY_METADATA_URL", "file://"+metadataPath)
	t.Setenv("STUF_CURRENCY_SEED_OUT", out)
	if err := runFromEnv(context.Background()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var rows []currencyseed.SeedCurrency
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
}
