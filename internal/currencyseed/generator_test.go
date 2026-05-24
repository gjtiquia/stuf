package currencyseed

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const ecbFixture = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
  <Cube>
    <Cube time="2026-05-22">
      <Cube currency="USD" rate="1.2"/>
      <Cube currency="JPY" rate="150"/>
      <Cube currency="HKD" rate="9.6"/>
      <Cube currency="XAU" rate="0.00036"/>
    </Cube>
  </Cube>
</gesmes:Envelope>`

const metadataFixture = `[
  {"code":"USD","name":"US Dollar","decimals":2},
  {"code":"EUR","name":"Euro","decimals":2},
  {"code":"JPY","name":"Yen","decimals":0},
  {"code":"HKD","name":"Hong Kong Dollar","decimals":2},
  {"code":"XAU","name":"Gold","decimals":5}
]`

func TestParseECBRatesAndMetadata(t *testing.T) {
	rates, err := ParseECBRates(strings.NewReader(ecbFixture))
	if err != nil {
		t.Fatal(err)
	}
	if rates.Date != "2026-05-22" {
		t.Fatalf("date = %q", rates.Date)
	}
	if got := rates.Rates["USD"].FloatString(1); got != "1.2" {
		t.Fatalf("USD rate = %s", got)
	}
	if got := rates.Rates["EUR"].FloatString(1); got != "1.0" {
		t.Fatalf("EUR implicit rate = %s", got)
	}
	metadata, err := ParseMetadata(strings.NewReader(metadataFixture))
	if err != nil {
		t.Fatal(err)
	}
	if metadata["JPY"].Name != "Yen" || metadata["JPY"].Decimals != 0 {
		t.Fatalf("bad JPY metadata: %+v", metadata["JPY"])
	}
}

func TestParseECBRatesRequiresUSDQuote(t *testing.T) {
	_, err := ParseECBRates(strings.NewReader(`<Cube><Cube time="2026-05-22"><Cube currency="JPY" rate="150"/></Cube></Cube>`))
	if err == nil || !strings.Contains(err.Error(), "missing USD") {
		t.Fatalf("expected missing USD error, got %v", err)
	}
}

func TestBuildSeedConvertsToUSDRelativeRates(t *testing.T) {
	rates, err := ParseECBRates(strings.NewReader(ecbFixture))
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := ParseMetadata(strings.NewReader(metadataFixture))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := BuildSeed(rates, metadata)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]SeedCurrency{}
	for _, row := range rows {
		got[row.Code] = row
	}
	if len(rows) != 4 {
		t.Fatalf("expected EUR, HKD, JPY, USD only, got %+v", rows)
	}
	if rows[0].Code != "EUR" || rows[1].Code != "HKD" || rows[2].Code != "JPY" || rows[3].Code != "USD" {
		t.Fatalf("rows not sorted by code: %+v", rows)
	}
	if got["USD"].RateToUSDAmount != 1 || got["USD"].RateToUSDScale != 0 {
		t.Fatalf("bad USD rate: %+v", got["USD"])
	}
	if got["EUR"].RateToUSDAmount != 12 || got["EUR"].RateToUSDScale != 1 {
		t.Fatalf("bad EUR rate: %+v", got["EUR"])
	}
	if got["HKD"].RateToUSDAmount != 125 || got["HKD"].RateToUSDScale != 3 {
		t.Fatalf("bad HKD rate: %+v", got["HKD"])
	}
	if got["JPY"].RateToUSDAmount != 8 || got["JPY"].RateToUSDScale != 3 || got["JPY"].Scale != 0 {
		t.Fatalf("bad JPY rate: %+v", got["JPY"])
	}
	if amount, scale, err := encodeRate(new(big.Rat).SetFrac64(1, 3)); err != nil || amount != 333333 || scale != maxRateScale {
		t.Fatalf("expected max %d decimal places, got amount=%d scale=%d err=%v", maxRateScale, amount, scale, err)
	}
	if _, ok := got["XAU"]; ok {
		t.Fatalf("metal code should be excluded: %+v", got["XAU"])
	}
}

func TestBuildSeedFailsOnMissingMetadata(t *testing.T) {
	rates, err := ParseECBRates(strings.NewReader(ecbFixture))
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := ParseMetadata(strings.NewReader(`[{"code":"USD","name":"US Dollar","decimals":2}]`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildSeed(rates, metadata); err == nil || !strings.Contains(err.Error(), "metadata missing") {
		t.Fatalf("expected metadata missing error, got %v", err)
	}
}

func TestRunFetchesAndWritesSeed(t *testing.T) {
	dir := t.TempDir()
	ecbPath := filepath.Join(dir, "ecb.xml")
	metadataPath := filepath.Join(dir, "metadata.json")
	out := filepath.Join(dir, "currencies.json")
	if err := os.WriteFile(ecbPath, []byte(ecbFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, []byte(metadataFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Config{
		ECBRatesURL:         "file://" + ecbPath,
		CurrencyMetadataURL: "file://" + metadataPath,
		OutputPath:          out,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var rows []SeedCurrency
	if err := json.Unmarshal(b, &rows); err != nil {
		t.Fatalf("generated seed is not compatible JSON: %v\n%s", err, b)
	}
	if len(rows) != 4 {
		t.Fatalf("rows = %+v", rows)
	}
}
