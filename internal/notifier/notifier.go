package notifier

import (
	"fmt"

	c "github.com/achannarasappa/ticker/v5/internal/common"
)

// Notifier is an interface for sending notifications.
type Notifier interface {
	Notify(message string)
}

// ConsoleNotifier sends notifications to the console.
type ConsoleNotifier struct{}

// Notify prints the message to the console.
func (n *ConsoleNotifier) Notify(message string) {
	fmt.Println(message)
}

// Alerter sends alerts based on price targets.
type Alerter struct {
	Notifier Notifier
}

type PriceAlertState struct {
	AboveTarget bool
	BelowTarget bool
}

// NewAlerter creates a new Alerter with the given notifier.
func NewAlerter(notifier Notifier) *Alerter {
	return &Alerter{Notifier: notifier}
}

// CheckAndNotify checks the asset for price target breaches and sends notifications.
func (a *Alerter) CheckAndNotify(asset c.Asset, currentState, newState PriceAlertState) {
	// if newState.AboveTarget && !currentState.AboveTarget {
	// 	message := fmt.Sprintf("ALERT: %s crossed above target price. Current price: %.2f",
	// 		asset.Symbol, asset.QuotePrice.Price)
	// 	a.Notifier.Notify(message)
	// }
	if newState.AboveTarget && !currentState.AboveTarget {
		message := fmt.Sprintf("ALERT: %s crossed above target price of %.2f. Current price: %.2f",
			asset.Symbol, *asset.Holding.TargetPriceAbove, asset.QuotePrice.Price)
		a.Notifier.Notify(message)
	}

	if newState.BelowTarget && !currentState.BelowTarget {
		message := fmt.Sprintf("ALERT: %s crossed below target price of %.2f. Current price: %.2f",
			asset.Symbol, *asset.Holding.TargetPriceBelow, asset.QuotePrice.Price)
		a.Notifier.Notify(message)
	}
}
