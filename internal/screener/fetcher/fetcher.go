package fetcher

import (
	"context"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	unary "github.com/achannarasappa/ticker/v5/internal/monitor/yahoo/unary"
)

// Fetcher is an interface for fetching asset quotes for a list of symbols.
type Fetcher interface {
	Fetch(ctx context.Context, symbols []string) ([]c.AssetQuote, error)
}

// YahooFetcher implements the Fetcher interface using the Yahoo Unary API.
type YahooFetcher struct {
	api *unary.UnaryAPI
}

// NewYahooFetcher creates a new YahooFetcher.
func NewYahooFetcher(api *unary.UnaryAPI) *YahooFetcher {
	return &YahooFetcher{api: api}
}

// Fetch retrieves quotes using the Yahoo API.
func (f *YahooFetcher) Fetch(ctx context.Context, symbols []string) ([]c.AssetQuote, error) {
	// The existing GetAssetQuotes function is perfect for this.
	quotes, _, err := f.api.GetAssetQuotes(symbols)
	return quotes, err
}
