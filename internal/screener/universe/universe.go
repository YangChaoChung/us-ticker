package universe

import (
	"encoding/json"
	"fmt"
	"os"
)

type Identifier string

// HoldingConfig defines a symbol and its specific target prices.
type HoldingConfig struct {
	Symbol           string   `json:"Symbol"`
	TargetPriceAbove *float64 `json:"TargetPriceAbove,omitempty"`
	TargetPriceBelow *float64 `json:"TargetPriceBelow,omitempty"`
}

const (
	Nasdaq100   Identifier = "nasdaq100"
	CryptoTop20 Identifier = "crypto_top20"
	MyWatchlist Identifier = "my_watchlist"
)

// For demonstration, we'll use hardcoded lists. This can be expanded.
var universeHoldings = map[Identifier][]HoldingConfig{
	Nasdaq100: {
		{Symbol: "AAPL"},
		{Symbol: "MSFT"},
		{Symbol: "AMZN"},
		{Symbol: "GOOGL"},
		{Symbol: "GOOG"},
		{Symbol: "NVDA"},
		{Symbol: "META"},
		{Symbol: "TSLA"},
	},
	CryptoTop20: {
		{Symbol: "BTC-USD"},
		{Symbol: "ETH-USD"},
		{Symbol: "SOL-USD"},
		{Symbol: "XRP-USD"},
		{Symbol: "DOGE-USD"},
	},
	// Default watchlist if the file is not found
	MyWatchlist: {
		{Symbol: "AAPL", TargetPriceAbove: floatp(200.00), TargetPriceBelow: floatp(150.00)},
		{Symbol: "MSFT", TargetPriceAbove: floatp(450.00), TargetPriceBelow: floatp(400.00)},
		{Symbol: "TSLA", TargetPriceBelow: floatp(170.00)},
	},
}

// RegisterUniverseFromFile loads a watchlist from a JSON file and registers it.
func RegisterUniverseFromFile(id Identifier, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("could not read watchlist file %s: %w", filePath, err)
	}

	var holdings []HoldingConfig
	if err := json.Unmarshal(data, &holdings); err != nil {
		return fmt.Errorf("could not parse watchlist file %s: %w", filePath, err)
	}

	universeHoldings[id] = holdings
	return nil
}

func GetHoldings(id Identifier) ([]HoldingConfig, error) {
	holdings, ok := universeHoldings[id]
	if !ok {
		// If the identifier is not a known universe, treat it as a single symbol with no targets.
		return []HoldingConfig{{Symbol: string(id)}}, nil
	}
	return holdings, nil
}

// GetSymbols is deprecated but kept for compatibility. Use GetHoldings instead.
func GetSymbols(id Identifier) ([]string, error) {
	holdings, err := GetHoldings(id)
	if err != nil {
		return nil, err
	}

	symbols := make([]string, len(holdings))
	for i, h := range holdings {
		symbols[i] = h.Symbol
	}
	return symbols, nil
}

// floatp is a helper to create a float64 pointer.
func floatp(v float64) *float64 {
	return &v
}
