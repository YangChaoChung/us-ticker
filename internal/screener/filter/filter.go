package filter

import (
	c "github.com/achannarasappa/ticker/v5/internal/common"
)

// Filter is an interface for a single screening criterion.
type Filter interface {
	Apply(quote c.AssetQuote) bool
}

// MarketCapFilter filters assets by market capitalization.
type MarketCapFilter struct {
	Min float64
	Max float64
}

func (f MarketCapFilter) Apply(quote c.AssetQuote) bool {
	if f.Min > 0 && quote.QuoteExtended.MarketCap < f.Min {
		return false
	}
	if f.Max > 0 && quote.QuoteExtended.MarketCap > f.Max {
		return false
	}
	return true
}

// PriceFilter filters assets by their price.
type PriceFilter struct {
	Min float64
	Max float64
}

func (f PriceFilter) Apply(quote c.AssetQuote) bool {
	if f.Min > 0 && quote.QuotePrice.Price < f.Min {
		return false
	}
	if f.Max > 0 && quote.QuotePrice.Price > f.Max {
		return false
	}
	return true
}
