package screener

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	c "github.com/achannarasappa/ticker/v5/internal/common"
	"github.com/achannarasappa/ticker/v5/internal/ui/util"
)

// Options to configure print behavior
type Options struct {
	Format string
}

type jsonRow struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func convertAssetsToCSV(assets []c.AssetQuote) string {
	rows := [][]string{
		{"name", "symbol", "price"},
	}

	for _, asset := range assets {
		rows = append(rows, []string{
			asset.Name,
			asset.Symbol,
			util.ConvertFloatToString(asset.QuotePrice.Price, true),
		})
	}

	b := new(bytes.Buffer)
	w := csv.NewWriter(b)
	//nolint:errcheck
	w.WriteAll(rows)

	return b.String()

}

func convertAssetsToJSON(assets []c.AssetQuote) string {
	var rows []jsonRow

	for _, asset := range assets {
		rows = append(rows, jsonRow{
			Name:   asset.Name,
			Symbol: asset.Symbol,
			Price:  fmt.Sprintf("%f", asset.QuotePrice.Price),
		})
	}

	if len(rows) == 0 {
		return "[]"
	}

	out, err := json.Marshal(rows)

	if err != nil {
		return err.Error()
	}

	return string(out)

}

// Print is a struct for printing output
type Print struct {
	w io.Writer
}

// NewPrint creates a new Print struct
func NewPrint(w io.Writer) *Print {
	return &Print{w: w}
}

// Render prints the assets to the writer
func (p *Print) Render(assets []c.AssetQuote, options Options) {
	if options.Format == "csv" {
		fmt.Fprintln(p.w, convertAssetsToCSV(assets))
		return
	}

	fmt.Fprintln(p.w, convertAssetsToJSON(assets))
}
