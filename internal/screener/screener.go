package screener

import (
	"context"
	"fmt"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/screener/fetcher"
	"github.com/achannarasappa/ticker/v5/internal/screener/filter"
	"github.com/achannarasappa/ticker/v5/internal/screener/universe"
)

// Screener orchestrates the screening process.
type Screener struct {
	fetcher fetcher.Fetcher
}

// NewScreener creates a new screener.
func NewScreener(f fetcher.Fetcher) *Screener {
	return &Screener{
		fetcher: f,
	}
}

// Config defines the configuration for a screen run.
type Config struct {
	Universe universe.Identifier
	Filters  []filter.Filter
}

// Run executes the screening process based on the provided config.
func (s *Screener) Run(ctx context.Context, config Config) ([]c.AssetQuote, error) {
	// 1. Get the list of holdings for the universe.
	holdings, err := universe.GetHoldings(config.Universe)
	if err != nil {
		return nil, fmt.Errorf("could not get holdings for universe %s: %w", config.Universe, err)
	}

	// Create a map for easy lookup of target prices by symbol.
	targetsBySymbol := make(map[string]universe.HoldingConfig)
	symbols := make([]string, len(holdings))
	for i, h := range holdings {
		symbols[i] = h.Symbol
		targetsBySymbol[h.Symbol] = h
	}

	// 2. Fetch quote data for all symbols.
	quotes, err := s.fetcher.Fetch(ctx, symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch quotes for screening: %w", err)
	}

	// 3. Apply filters and merge target prices.
	var screenedQuotes []c.AssetQuote
	for _, quote := range quotes {
		// Apply per-symbol target prices from the universe
		if holdingConfig, ok := targetsBySymbol[quote.Symbol]; ok {
			quote.TargetPriceAbove = holdingConfig.TargetPriceAbove
			quote.TargetPriceBelow = holdingConfig.TargetPriceBelow
		}

		passesAllFilters := true
		for _, f := range config.Filters {
			if !f.Apply(quote) {
				passesAllFilters = false
				break
			}
		}
		if passesAllFilters {
			screenedQuotes = append(screenedQuotes, quote)
		}
	}

	return screenedQuotes, nil
}
