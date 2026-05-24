package main

import (
	"context"
	"fmt"
	"os"

	"stuf/internal/currencyseed"
)

func main() {
	if err := runFromEnv(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "refresh currencies: %v\n", err)
		os.Exit(1)
	}
}

func runFromEnv(ctx context.Context) error {
	return currencyseed.Run(ctx, currencyseed.Config{
		ECBRatesURL:         os.Getenv("STUF_ECB_RATES_URL"),
		CurrencyMetadataURL: os.Getenv("STUF_CURRENCY_METADATA_URL"),
		OutputPath:          os.Getenv("STUF_CURRENCY_SEED_OUT"),
	})
}
