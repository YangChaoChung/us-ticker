package asset

import (
	c "github.com/achannarasappa/ticker/v5/internal/common"
)

// SetTargetPrices sets the target prices for a list of assets.
func SetTargetPrices(assets []c.AssetQuote, targetPriceAbove, targetPriceBelow float64) []c.AssetQuote {
	if targetPriceAbove <= 0 && targetPriceBelow <= 0 {
		return assets
	}

	for i := range assets {
		if targetPriceAbove > 0 {
			assets[i].TargetPriceAbove = &targetPriceAbove
		}
		if targetPriceBelow > 0 {
			assets[i].TargetPriceBelow = &targetPriceBelow
		}
	}
	return assets
}
