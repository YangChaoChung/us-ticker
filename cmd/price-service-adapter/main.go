package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	unary "github.com/achannarasappa/ticker/v5/internal/monitor/yahoo/unary"
)

type rawQuote struct {
	InputTicker string `json:"input_ticker"`
	Timestamp   string `json:"timestamp"`
	c.AssetQuote
}

func main() {
	tickerList := flag.String("tickers", "", "comma-separated ticker symbols")
	flag.Parse()

	tickers := parseTickers(*tickerList)
	if len(tickers) == 0 {
		fmt.Fprintln(os.Stderr, "no tickers provided")
		os.Exit(2)
	}

	api := unary.NewUnaryAPI(unary.Config{
		BaseURL:           "https://query1.finance.yahoo.com",
		SessionRootURL:    "https://finance.yahoo.com",
		SessionCrumbURL:   "https://query2.finance.yahoo.com",
		SessionConsentURL: "https://consent.yahoo.com",
	})

	_, quotesBySymbol, err := api.GetAssetQuotes(tickers)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result := make([]rawQuote, 0, len(quotesBySymbol))
	for _, ticker := range tickers {
		quote, ok := quotesBySymbol[ticker]
		if !ok {
			continue
		}

		result = append(result, rawQuote{
			InputTicker: ticker,
			Timestamp:   now,
			AssetQuote:  *quote,
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseTickers(value string) []string {
	tickers := []string{}
	seen := map[string]bool{}
	for _, item := range strings.Split(value, ",") {
		ticker := strings.ToUpper(strings.TrimSpace(item))
		if ticker == "" || seen[ticker] {
			continue
		}
		seen[ticker] = true
		tickers = append(tickers, ticker)
	}

	return tickers
}
