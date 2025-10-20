package asset

import (
	"bytes"
	"strings"
	"testing"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/notifier"
)

func TestProcessAssetsPriceAlerts(t *testing.T) {
	targetPriceAbove := 100.0
	targetPriceBelow := 80.0
	buffer := &bytes.Buffer{}

	priceAlertMu.Lock()
	originalWriter := priceAlertWriter
	originalStates := priceAlertStates
	priceAlertWriter = buffer
	priceAlertStates = make(map[string]notifier.PriceAlertState)
	priceAlertMu.Unlock()

	defer func() {
		priceAlertMu.Lock()
		priceAlertWriter = originalWriter
		priceAlertStates = originalStates
		priceAlertMu.Unlock()
	}()

	asset := c.Asset{
		Symbol: "ABC",
		Holding: c.Holding{
			TargetPriceAbove: &targetPriceAbove,
			TargetPriceBelow: &targetPriceBelow,
		},
		QuotePrice: c.QuotePrice{Price: 90.0},
	}

	ProcessAssets([]c.Asset{asset})
	if buffer.Len() != 0 {
		t.Fatalf("expected no alert on initial evaluation, got %q", buffer.String())
	}

	asset.QuotePrice.Price = 110.0
	ProcessAssets([]c.Asset{asset})
	output := buffer.String()
	if !strings.Contains(output, "ALERT: ABC crossed above target price of 100.00. Current price: 110.00") {
		t.Fatalf("expected alert to mention symbol, current price, and target price, got %q", output)
	}

	buffer.Reset()
	asset.QuotePrice.Price = 120.0
	ProcessAssets([]c.Asset{asset})
	if buffer.Len() != 0 {
		t.Fatalf("expected no repeated alert while price remains above target, got %q", buffer.String())
	}

	asset.QuotePrice.Price = 70.0
	ProcessAssets([]c.Asset{asset})
	output = buffer.String()
	if !strings.Contains(output, "ABC") || !strings.Contains(output, "70.00") || !strings.Contains(output, "80.00") {
		t.Fatalf("expected alert for crossing below target, got %q", output)
	}

	buffer.Reset()
	asset.QuotePrice.Price = 60.0
	ProcessAssets([]c.Asset{asset})
	if buffer.Len() != 0 {
		t.Fatalf("expected no repeated alert while price remains below target, got %q", buffer.String())
	}

	asset.QuotePrice.Price = 95.0
	ProcessAssets([]c.Asset{asset})
	buffer.Reset()

	asset.QuotePrice.Price = 103.0
	ProcessAssets([]c.Asset{asset})
	output = buffer.String()
	if !strings.Contains(output, "ABC") || !strings.Contains(output, "103.00") {
		t.Fatalf("expected alert after price moves above target again, got %q", output)
	}
}
