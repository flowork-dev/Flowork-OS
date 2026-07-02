// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN — jangan edit file ini. 📄 Dok: lock/tools.md
// Nambah/ubah tool TANPA buka frozen: pakai SEAM tools.Register() di file sibling non-frozen.
// market.go — the investment team's "eyes" exposed as an engine tool.
//
// Agents cannot reach the /api/market/quote HTTP endpoint (the auth middleware
// gates it and its exempt list is frozen), so the same data is offered here as a
// registered tool: a loket-native agent calls it via the blessed `tool.run`
// capability (SandboxRunV3 enforces net:fetch + consent on top, never a bypass).
// Both surfaces share marketdata.Fetch — one place knows the upstream + its auth.
package builtins

import (
	"context"

	"flowork-gui/internal/marketdata"
	"flowork-gui/internal/tools"
)

func init() { tools.Register(&marketQuoteTool{}) }

type marketQuoteTool struct{}

func (marketQuoteTool) Name() string       { return "market_quote" }
func (marketQuoteTool) Capability() string { return "net:fetch:*" }
func (marketQuoteTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Fetch real market data for ONE stock ticker: spot price, recent OHLCV candles (for technical analysis) and key fundamentals (PE, PB, ROE, margins, revenue, debt). IDX (Indonesia) tickers take the .JK suffix, e.g. BBCA.JK; global tickers use their native symbol, e.g. AAPL.",
		Params: []tools.Param{
			{Name: "ticker", Type: tools.ParamString, Description: "ticker symbol, e.g. BBCA.JK or AAPL", Required: true},
			{Name: "range", Type: tools.ParamString, Description: "candle history window (default 6mo): 1mo, 3mo, 6mo, 1y, 2y, 5y", Required: false},
			{Name: "interval", Type: tools.ParamString, Description: "candle interval (default 1d): 1d, 1wk, 1mo", Required: false},
		},
		Returns: "{ticker, price, currency, exchange, ohlcv:[{t,o,h,l,c,v}], fundamentals:{pe,pb,roe,profit_margin,revenue,debt_to_equity,...}}",
	}
}

func (marketQuoteTool) Run(_ context.Context, args map[string]any) (tools.Result, error) {
	ticker, _ := args["ticker"].(string)
	rng, _ := args["range"].(string)
	ivl, _ := args["interval"].(string)
	out, err := marketdata.Fetch(ticker, rng, ivl)
	if err != nil {
		return tools.Result{}, err
	}
	return tools.Result{Output: out}, nil
}
