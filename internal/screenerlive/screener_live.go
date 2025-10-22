package screenerlive

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/monitor"
	yahooUnary "github.com/achannarasappa/ticker/v5/internal/monitor/yahoo/unary"
	"github.com/achannarasappa/ticker/v5/internal/screener"
	"github.com/achannarasappa/ticker/v5/internal/screener/fetcher"
	"github.com/achannarasappa/ticker/v5/internal/screener/filter"
	"github.com/achannarasappa/ticker/v5/internal/screener/universe"
)

const defaultRefreshInterval = 15 * time.Second

// Update represents a live quote update emitted by the live screener.
type Update struct {
	Quote         c.AssetQuote
	VersionVector int
}

// Result contains the initial snapshot and channels for streaming updates or errors.
type Result struct {
	Initial []c.AssetQuote
	Updates <-chan Update
	Errors  <-chan error
	Stop    func()
}

// Config controls the live screen run.
type Config struct {
	Universe universe.Identifier
	Filters  []filter.Filter
}

// LiveScreener orchestrates a traditional screener with live quote tracking.
type LiveScreener struct {
	screener *screener.Screener
	dep      c.Dependencies
	refresh  time.Duration
}

// New creates a LiveScreener using the provided dependencies and refresh cadence.
func New(dep c.Dependencies, refresh time.Duration) *LiveScreener {
	if refresh <= 0 {
		refresh = defaultRefreshInterval
	}

	yahooAPI := yahooUnary.NewUnaryAPI(yahooUnary.Config{
		BaseURL:           dep.MonitorYahooBaseURL,
		SessionRootURL:    dep.MonitorYahooSessionRootURL,
		SessionCrumbURL:   dep.MonitorYahooSessionCrumbURL,
		SessionConsentURL: dep.MonitorYahooSessionConsentURL,
	})

	return &LiveScreener{
		screener: screener.NewScreener(fetcher.NewYahooFetcher(yahooAPI)),
		dep:      dep,
		refresh:  refresh,
	}
}

// Run executes the screen and begins streaming live updates for the matching symbols.
func (ls *LiveScreener) Run(ctx context.Context, cfg Config) (Result, error) {
	quotes, err := ls.screener.Run(ctx, screener.Config{Universe: cfg.Universe, Filters: cfg.Filters})
	if err != nil {
		return Result{}, fmt.Errorf("failed to run base screener: %w", err)
	}

	if len(quotes) == 0 {
		closedUpdates := make(chan Update)
		close(closedUpdates)
		closedErrors := make(chan error)
		close(closedErrors)

		return Result{
			Initial: []c.AssetQuote{},
			Updates: closedUpdates,
			Errors:  closedErrors,
			Stop:    func() {},
		}, nil
	}

	symbols := make([]string, 0, len(quotes))
	for _, quote := range quotes {
		symbols = append(symbols, quote.Symbol)
	}

	stockSymbols, cryptoSymbols := partitionSymbols(symbols)

	sources := make([]c.AssetGroupSymbolsBySource, 0, 2)
	if len(stockSymbols) > 0 {
		sources = append(sources, c.AssetGroupSymbolsBySource{Symbols: stockSymbols, Source: c.QuoteSourceYahoo})
	}
	if len(cryptoSymbols) > 0 {
		sources = append(sources, c.AssetGroupSymbolsBySource{Symbols: cryptoSymbols, Source: c.QuoteSourceCoinbase})
	}

	if len(sources) == 0 {
		// Fallback – treat everything as Yahoo quotes if we cannot categorise.
		sources = append(sources, c.AssetGroupSymbolsBySource{Symbols: symbols, Source: c.QuoteSourceYahoo})
	}

	errChan := make(chan error, 16)
	updates := make(chan Update, 256)
	done := make(chan struct{})
	stopOnce := sync.Once{}

	refreshSeconds := int(ls.refresh.Seconds())
	if refreshSeconds <= 0 {
		refreshSeconds = int(defaultRefreshInterval.Seconds())
	}

	logger := log.New(logChannelWriter{ch: errChan}, "", 0)

	mon, err := monitor.NewMonitor(monitor.ConfigMonitor{
		RefreshInterval: refreshSeconds,
		TargetCurrency:  "USD",
		Logger:          logger,
		ConfigMonitorPriceCoinbase: monitor.ConfigMonitorPriceCoinbase{
			BaseURL:      ls.dep.MonitorPriceCoinbaseBaseURL,
			StreamingURL: ls.dep.MonitorPriceCoinbaseStreamingURL,
		},
		ConfigMonitorsYahoo: monitor.ConfigMonitorsYahoo{
			BaseURL:           ls.dep.MonitorYahooBaseURL,
			SessionRootURL:    ls.dep.MonitorYahooSessionRootURL,
			SessionCrumbURL:   ls.dep.MonitorYahooSessionCrumbURL,
			SessionConsentURL: ls.dep.MonitorYahooSessionConsentURL,
		},
	})
	if err != nil {
		return Result{}, fmt.Errorf("failed to create live monitor: %w", err)
	}

	currentVersion := int(time.Now().UnixNano())

	if err := mon.SetOnUpdate(monitor.ConfigUpdateFns{
		OnUpdateAssetQuote: func(_ string, assetQuote c.AssetQuote, versionVector int) {
			if versionVector != currentVersion {
				return
			}

			select {
			case <-done:
				return
			default:
			}

			select {
			case updates <- Update{Quote: assetQuote, VersionVector: versionVector}:
			default:
			}
		},
		OnUpdateAssetGroupQuote: func(assetGroupQuote c.AssetGroupQuote, versionVector int) {
			if versionVector != currentVersion {
				return
			}

			select {
			case <-done:
				return
			default:
			}

			for _, quote := range assetGroupQuote.AssetQuotes {
				select {
				case updates <- Update{Quote: quote, VersionVector: versionVector}:
				default:
				}
			}
		},
	}); err != nil {
		mon.Stop()

		return Result{}, fmt.Errorf("failed to configure live monitor callbacks: %w", err)
	}

	mon.Start()

	assetGroup := c.AssetGroup{
		ConfigAssetGroup: c.ConfigAssetGroup{
			Name:      "live-screen",
			Watchlist: append([]string(nil), symbols...),
		},
		SymbolsBySource: sources,
	}

	if err := mon.SetAssetGroup(assetGroup, currentVersion); err != nil {
		mon.Stop()

		return Result{}, fmt.Errorf("failed to start live monitoring: %w", err)
	}

	initial := mon.GetAssetGroupQuote().AssetQuotes

	// Create a lookup for the target prices from the initial screener run.
	targetsBySymbol := make(map[string]c.AssetQuote)
	for _, q := range quotes {
		targetsBySymbol[q.Symbol] = q
	}

	// Merge the target prices into the initial results from the monitor.
	for i, q := range initial {
		if targets, ok := targetsBySymbol[q.Symbol]; ok {
			initial[i].TargetPriceAbove = targets.TargetPriceAbove
			initial[i].TargetPriceBelow = targets.TargetPriceBelow
		}
	}

	stopFn := func() {
		stopOnce.Do(func() {
			close(done)
			mon.Stop()
			close(updates)
			close(errChan)
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			stopFn()
		case <-done:
		}
	}()

	return Result{
		Initial: initial,
		Updates: updates,
		Errors:  errChan,
		Stop:    stopFn,
	}, nil
}

func partitionSymbols(symbols []string) (stocks []string, crypto []string) {
	for _, symbol := range symbols {
		if isLikelyCrypto(symbol) {
			crypto = append(crypto, symbol)
		} else {
			stocks = append(stocks, symbol)
		}
	}

	return stocks, crypto
}

func isLikelyCrypto(symbol string) bool {
	upper := strings.ToUpper(symbol)

	return strings.Contains(upper, "-USD") || strings.Contains(upper, "-USDT")
}

type logChannelWriter struct {
	ch chan<- error
}

func (w logChannelWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}

	select {
	case w.ch <- fmt.Errorf("%s", msg):
	default:
	}

	return len(p), nil
}
