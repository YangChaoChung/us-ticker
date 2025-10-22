package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/achannarasappa/ticker/v5/internal/asset"
	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/monitor/yahoo/unary"
	"github.com/achannarasappa/ticker/v5/internal/screener"
	"github.com/achannarasappa/ticker/v5/internal/screener/fetcher"
	"github.com/achannarasappa/ticker/v5/internal/screener/filter"
	"github.com/achannarasappa/ticker/v5/internal/screener/universe"
	"github.com/achannarasappa/ticker/v5/internal/screenerlive"
	"github.com/spf13/cobra"
)

var (
	screenerUniverse           string
	screenerMinMarketCap       float64
	screenerMaxMarketCap       float64
	screenerMinPrice           float64
	screenerMaxPrice           float64
	screenLive                 bool
	screenLiveTargetPriceAbove float64
	screenLiveTargetPriceBelow float64
	screenLiveRefresh          int
	screenerWatchlistFile      string
)

// screenCmd represents the screen command
var screenCmd = &cobra.Command{
	Use:   "screen",
	Short: "Screen for assets and optionally stream live price updates",
	Long: `Screen for assets from a universe (e.g., nasdaq100) with filters.
The --live flag will continuously monitor prices and trigger notifications.
Example (one-off screen):
ticker screen --universe nasdaq100 --min-market-cap 100000000000

Example (live monitoring):
ticker screen --universe=AAPL --target-price-above=200.00 --target-price-below=120.00 --live`,
	Run: func(cmd *cobra.Command, args []string) {
		if screenerWatchlistFile != "" {
			if err := universe.RegisterUniverseFromFile(universe.MyWatchlist, screenerWatchlistFile); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load watchlist file: %v\n", err)
			}
		}

		if screenLive {
			runLiveScreener(cmd, args)
		} else {
			runScreener(cmd, args)
		}
	},
}

func runScreener(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Initialize the Yahoo API
	yahooAPI := unary.NewUnaryAPI(unary.Config{
		BaseURL:           dep.MonitorYahooBaseURL,
		SessionRootURL:    dep.MonitorYahooSessionRootURL,
		SessionCrumbURL:   dep.MonitorYahooSessionCrumbURL,
		SessionConsentURL: dep.MonitorYahooSessionConsentURL,
	})

	// Initialize the fetcher
	yahooFetcher := fetcher.NewYahooFetcher(yahooAPI)

	// Initialize the screener
	s := screener.NewScreener(yahooFetcher)

	// Build filters from flags
	var filters []filter.Filter
	if screenerMinMarketCap > 0 || screenerMaxMarketCap > 0 {
		filters = append(filters, filter.MarketCapFilter{Min: screenerMinMarketCap, Max: screenerMaxMarketCap})
	}
	if screenerMinPrice > 0 || screenerMaxPrice > 0 {
		filters = append(filters, filter.PriceFilter{Min: screenerMinPrice, Max: screenerMaxPrice})
	}

	// Configure and run the screener
	config := screener.Config{
		Universe: universe.Identifier(screenerUniverse),
		Filters:  filters,
	}

	quotes, err := s.Run(ctx, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running screener: %v\n", err)
		os.Exit(1)
	}

	// Print the results
	p := screener.NewPrint(os.Stdout)
	p.Render(quotes, screener.Options{Format: optionsPrint.Format})
}

func runLiveScreener(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	refreshDuration := time.Duration(screenLiveRefresh) * time.Second
	live := screenerlive.New(dep, refreshDuration)

	var filters []filter.Filter
	if screenerMinMarketCap > 0 || screenerMaxMarketCap > 0 {
		filters = append(filters, filter.MarketCapFilter{Min: screenerMinMarketCap, Max: screenerMaxMarketCap})
	}
	if screenerMinPrice > 0 || screenerMaxPrice > 0 {
		filters = append(filters, filter.PriceFilter{Min: screenerMinPrice, Max: screenerMaxPrice})
	}

	result, err := live.Run(ctx, screenerlive.Config{
		Universe: universe.Identifier(screenerUniverse),
		Filters:  filters,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running live screener: %v\n", err)
		os.Exit(1)
	}
	defer result.Stop()

	// Target prices are now set by the screener from the universe definition.
	// The global flags are still used for ad-hoc single-symbol screening.
	if len(result.Initial) == 1 {
		result.Initial = asset.SetTargetPrices(result.Initial, screenLiveTargetPriceAbove, screenLiveTargetPriceBelow)
	}

	// Process initial assets for alerts
	initialAssetsForAlerts := make([]c.Asset, len(result.Initial))
	for i, quote := range result.Initial {
		initialAssetsForAlerts[i] = c.Asset{
			Symbol:     quote.Symbol,
			QuotePrice: quote.QuotePrice,
			Holding: c.Holding{
				TargetPriceAbove: quote.TargetPriceAbove,
				TargetPriceBelow: quote.TargetPriceBelow,
			},
		}
	}
	asset.ProcessAssets(initialAssetsForAlerts)

	if len(result.Initial) == 0 {
		fmt.Fprintln(os.Stdout, "No symbols matched the provided filters.")
		return
	}

	printer := screener.NewPrint(os.Stdout)

	current := make(map[string]c.AssetQuote, len(result.Initial))
	for _, quote := range result.Initial {
		current[quote.Symbol] = quote
	}

	fmt.Fprintf(os.Stdout, "Initial snapshot (%d symbols)\n", len(result.Initial))
	printer.Render(result.Initial, screener.Options{Format: optionsPrint.Format})

	renderInterval := refreshDuration
	if renderInterval <= 0 {
		renderInterval = 30 * time.Second
	}

	snapshotTicker := time.NewTicker(renderInterval)
	defer snapshotTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stdout, "Live screen stopped.")
			return
		case err, ok := <-result.Errors:
			if ok && err != nil {
				fmt.Fprintf(os.Stderr, "live screen error: %v\n", err)
			}
		case update, ok := <-result.Updates:
			if !ok {
				return
			}

			// Get the existing quote to retrieve target prices from the initial universe/watchlist load.
			existingQuote, ok := current[update.Quote.Symbol]
			if !ok {
				// This should not happen if the initial snapshot was processed correctly, but we'll skip if it does.
				continue
			}

			// Apply ad-hoc target prices if set via flags for a single symbol. This overrides universe/watchlist targets.
			if len(current) == 1 {
				if screenLiveTargetPriceAbove > 0 {
					existingQuote.TargetPriceAbove = &screenLiveTargetPriceAbove
				}
				if screenLiveTargetPriceBelow > 0 {
					existingQuote.TargetPriceBelow = &screenLiveTargetPriceBelow
				}
			}

			// Create an asset for alert processing with the new price but the original target prices.
			asset.ProcessAssets([]c.Asset{
				{
					Symbol:     update.Quote.Symbol,
					QuotePrice: update.Quote.QuotePrice,
					Holding: c.Holding{
						TargetPriceAbove: existingQuote.TargetPriceAbove,
						TargetPriceBelow: existingQuote.TargetPriceBelow,
					},
				},
			})

			// Update the current quote with the new price information for the next snapshot render.
			existingQuote.QuotePrice = update.Quote.QuotePrice
			current[update.Quote.Symbol] = existingQuote

		case <-snapshotTicker.C:
			fmt.Fprintf(os.Stdout, "\nSnapshot @ %s\n", time.Now().Format(time.RFC3339))

			quotes := make([]c.AssetQuote, 0, len(current))
			for _, quote := range current {
				quotes = append(quotes, quote)
			}

			printer.Render(quotes, screener.Options{Format: optionsPrint.Format})
		}
	}
}

func init() {
	rootCmd.AddCommand(screenCmd)

	screenCmd.Flags().StringVar(&screenerUniverse, "universe", "nasdaq100", "Asset universe to screen (e.g., nasdaq100, crypto_top20)")
	screenCmd.Flags().Float64Var(&screenerMinMarketCap, "min-market-cap", 0, "Minimum market cap")
	screenCmd.Flags().Float64Var(&screenerMaxMarketCap, "max-market-cap", 0, "Maximum market cap")
	screenCmd.Flags().Float64Var(&screenerMinPrice, "min-price", 0, "Minimum price")
	screenCmd.Flags().Float64Var(&screenerMaxPrice, "max-price", 0, "Maximum price")
	screenCmd.Flags().BoolVar(&screenLive, "live", false, "Enable live price streaming")
	screenCmd.Flags().Float64Var(&screenLiveTargetPriceAbove, "target-price-above", 0, "Target price for above alerts (requires --live)")
	screenCmd.Flags().Float64Var(&screenLiveTargetPriceBelow, "target-price-below", 0, "Target price for below alerts (requires --live)")
	screenCmd.Flags().IntVar(&screenLiveRefresh, "refresh", 15, "Refresh interval in seconds (requires --live)")
	screenCmd.Flags().StringVar(&screenerWatchlistFile, "watchlist-file", "", "path to a JSON file to define a custom universe")
}
