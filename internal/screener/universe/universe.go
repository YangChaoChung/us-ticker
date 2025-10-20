package universe

type Identifier string

const (
	Nasdaq100   Identifier = "nasdaq100"
	CryptoTop20 Identifier = "crypto_top20"
)

// For demonstration, we'll use hardcoded lists. This can be expanded.
var universeSymbols = map[Identifier][]string{
	Nasdaq100:   {"AAPL", "MSFT", "AMZN", "GOOGL", "GOOG", "NVDA", "META", "TSLA"}, // Example subset
	CryptoTop20: {"BTC-USD", "ETH-USD", "SOL-USD", "XRP-USD", "DOGE-USD"},          // Example subset
}

func GetSymbols(id Identifier) ([]string, error) {
	symbols, ok := universeSymbols[id]
	if !ok {
		// If the identifier is not a known universe, treat it as a single symbol.
		return []string{string(id)}, nil
	}
	return symbols, nil
}
