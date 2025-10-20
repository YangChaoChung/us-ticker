package asset

import (
	"fmt"
	"io"
	"os"
	"sync"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/notifier"
)

var (
	priceAlertWriter io.Writer = os.Stdout
	priceAlertStates           = make(map[string]notifier.PriceAlertState)
	priceAlertMu     sync.Mutex
	alerter          = notifier.NewAlerter(&notifier.ConsoleNotifier{})
)

// SetPriceAlertWriter sets the output writer for price alerts.
func SetPriceAlertWriter(w io.Writer) {
	priceAlertMu.Lock()
	defer priceAlertMu.Unlock()
	priceAlertWriter = w
}

// ProcessAssets checks assets for price alert conditions and prints notifications.
func ProcessAssets(assets []c.Asset) {
	priceAlertMu.Lock()
	defer priceAlertMu.Unlock()

	for _, asset := range assets {
		if asset.Holding.TargetPriceAbove == nil && asset.Holding.TargetPriceBelow == nil {
			continue
		}

		currentState, exists := priceAlertStates[asset.Symbol]
		var isAboveTarget, isBelowTarget bool

		if asset.Holding.TargetPriceAbove != nil {
			isAboveTarget = asset.QuotePrice.Price >= *asset.Holding.TargetPriceAbove
		}

		if asset.Holding.TargetPriceBelow != nil {
			isBelowTarget = asset.QuotePrice.Price <= *asset.Holding.TargetPriceBelow
		}

		newState := notifier.PriceAlertState{AboveTarget: isAboveTarget, BelowTarget: isBelowTarget}

		// If the state doesn't exist, this is the first check.
		// We check if the new state itself warrants an alert.
		if !exists {
			if newState.AboveTarget {
				fmt.Fprintf(priceAlertWriter, "ALERT: %s is above target price of %.2f. Current price: %.2f\n",
					asset.Symbol, *asset.Holding.TargetPriceAbove, asset.QuotePrice.Price)
			}
			if newState.BelowTarget {
				fmt.Fprintf(priceAlertWriter, "ALERT: %s is below target price of %.2f. Current price: %.2f\n",
					asset.Symbol, *asset.Holding.TargetPriceBelow, asset.QuotePrice.Price)
			}
			priceAlertStates[asset.Symbol] = newState
			continue
		}

		// For existing states, check for a state change (crossing the threshold)
		alerter.CheckAndNotify(asset, currentState, newState) // CC CHECK
		// print the state changes
		if newState.AboveTarget && !currentState.AboveTarget {
			fmt.Fprintf(priceAlertWriter, "ALERT: %s crossed above target price of %.2f. Current price: %.2f\n",
				asset.Symbol, *asset.Holding.TargetPriceAbove, asset.QuotePrice.Price)
		}
		if newState.BelowTarget && !currentState.BelowTarget {
			fmt.Fprintf(priceAlertWriter, "ALERT: %s crossed below target price of %.2f. Current price: %.2f\n",
				asset.Symbol, *asset.Holding.TargetPriceBelow, asset.QuotePrice.Price)
		}

		priceAlertStates[asset.Symbol] = newState
	}
}
