package currencyseed

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultECBRatesURL         = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	DefaultCurrencyMetadataURL = "https://staticdata.dev/v1/currencies.json"
	DefaultOutputPath          = "internal/seed/currencies.json"
	maxRateScale               = 6
)

type Config struct {
	ECBRatesURL         string
	CurrencyMetadataURL string
	OutputPath          string
	Client              *http.Client
}

type SeedCurrency struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	Scale           int    `json:"scale"`
	RateToUSDAmount int64  `json:"rate_to_usd_amount"`
	RateToUSDScale  int    `json:"rate_to_usd_scale"`
}

type ECBRates struct {
	Date  string
	Rates map[string]*big.Rat
}

type Metadata struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Decimals int    `json:"decimals"`
}

func Run(ctx context.Context, cfg Config) error {
	cfg = cfg.withDefaults()
	ecbBody, err := fetch(ctx, cfg.Client, cfg.ECBRatesURL)
	if err != nil {
		return fmt.Errorf("fetch ECB rates: %w", err)
	}
	metaBody, err := fetch(ctx, cfg.Client, cfg.CurrencyMetadataURL)
	if err != nil {
		return fmt.Errorf("fetch currency metadata: %w", err)
	}
	rates, err := ParseECBRates(bytes.NewReader(ecbBody))
	if err != nil {
		return err
	}
	metadata, err := ParseMetadata(bytes.NewReader(metaBody))
	if err != nil {
		return err
	}
	rows, err := BuildSeed(rates, metadata)
	if err != nil {
		return err
	}
	b, err := MarshalSeed(rows)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfg.OutputPath, b, 0o644); err != nil {
		return fmt.Errorf("write currency seed: %w", err)
	}
	return nil
}

func (c Config) withDefaults() Config {
	if c.ECBRatesURL == "" {
		c.ECBRatesURL = DefaultECBRatesURL
	}
	if c.CurrencyMetadataURL == "" {
		c.CurrencyMetadataURL = DefaultCurrencyMetadataURL
	}
	if c.OutputPath == "" {
		c.OutputPath = DefaultOutputPath
	}
	if c.Client == nil {
		c.Client = &http.Client{Timeout: 30 * time.Second}
	}
	return c
}

func fetch(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	if strings.HasPrefix(url, "file://") {
		return os.ReadFile(strings.TrimPrefix(url, "file://"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s returned %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func ParseECBRates(r io.Reader) (ECBRates, error) {
	dec := xml.NewDecoder(r)
	out := ECBRates{Rates: map[string]*big.Rat{"EUR": big.NewRat(1, 1)}}
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return ECBRates{}, fmt.Errorf("parse ECB XML: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "Cube" {
			continue
		}
		var currency, rate, date string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "time":
				date = attr.Value
			case "currency":
				currency = strings.ToUpper(attr.Value)
			case "rate":
				rate = attr.Value
			}
		}
		if date != "" {
			out.Date = date
		}
		if currency == "" && rate == "" {
			continue
		}
		if currency == "" || rate == "" {
			return ECBRates{}, errors.New("ECB rate row missing currency or rate")
		}
		value, ok := new(big.Rat).SetString(rate)
		if !ok {
			return ECBRates{}, fmt.Errorf("invalid ECB rate for %s: %q", currency, rate)
		}
		out.Rates[currency] = value
	}
	if out.Date == "" {
		return ECBRates{}, errors.New("ECB rates missing date")
	}
	if _, ok := out.Rates["USD"]; !ok {
		return ECBRates{}, errors.New("ECB rates missing USD quote")
	}
	return out, nil
}

func ParseMetadata(r io.Reader) (map[string]Metadata, error) {
	var rows []Metadata
	if err := json.NewDecoder(r).Decode(&rows); err != nil {
		return nil, fmt.Errorf("parse currency metadata: %w", err)
	}
	out := make(map[string]Metadata, len(rows))
	for _, row := range rows {
		row.Code = strings.ToUpper(strings.TrimSpace(row.Code))
		if row.Code == "" {
			continue
		}
		out[row.Code] = row
	}
	return out, nil
}

func BuildSeed(rates ECBRates, metadata map[string]Metadata) ([]SeedCurrency, error) {
	usdPerEUR := rates.Rates["USD"]
	if usdPerEUR == nil {
		return nil, errors.New("ECB rates missing USD quote")
	}
	var codes []string
	for code := range rates.Rates {
		if !isFiatCode(code) {
			continue
		}
		codes = append(codes, code)
	}
	sort.Strings(codes)
	out := make([]SeedCurrency, 0, len(codes))
	for _, code := range codes {
		meta, ok := metadata[code]
		if !ok {
			return nil, fmt.Errorf("metadata missing for currency %s", code)
		}
		rate := big.NewRat(1, 1)
		switch code {
		case "USD":
			rate = big.NewRat(1, 1)
		case "EUR":
			rate = new(big.Rat).Set(usdPerEUR)
		default:
			rate = new(big.Rat).Quo(usdPerEUR, rates.Rates[code])
		}
		amount, scale, err := encodeRate(rate)
		if err != nil {
			return nil, fmt.Errorf("encode rate for %s: %w", code, err)
		}
		out = append(out, SeedCurrency{
			Code:            code,
			Name:            meta.Name,
			Scale:           meta.Decimals,
			RateToUSDAmount: amount,
			RateToUSDScale:  scale,
		})
	}
	return out, nil
}

func isFiatCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	if strings.HasPrefix(code, "X") && code != "XCD" && code != "XOF" && code != "XAF" && code != "XPF" {
		return false
	}
	return true
}

func encodeRate(rate *big.Rat) (int64, int, error) {
	if rate == nil || rate.Sign() <= 0 {
		return 0, 0, errors.New("rate must be positive")
	}
	decimal := rate.FloatString(maxRateScale)
	decimal = strings.TrimRight(decimal, "0")
	decimal = strings.TrimRight(decimal, ".")
	if decimal == "" {
		decimal = "0"
	}
	parts := strings.SplitN(decimal, ".", 2)
	scale := 0
	digits := parts[0]
	if len(parts) == 2 {
		scale = len(parts[1])
		digits += parts[1]
	}
	amount, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return amount, scale, nil
}

func MarshalSeed(rows []SeedCurrency) ([]byte, error) {
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
