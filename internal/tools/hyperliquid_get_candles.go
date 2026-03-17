package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
)

func newHyperliquidGetCandlesTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_candles",
		mcp.WithDescription(
			"Query OHLCV candle data for a coin on Hyperliquid. "+
				"Returns up to 5000 candles with open, high, low, close, volume, and trade count. "+
				"Rate-limit weight scales with returned item count — prefer narrow time ranges in agentic loops.",
		),
		mcp.WithString("coin",
			mcp.Description("Coin symbol (e.g. BTC, ETH)."),
			mcp.Required(),
		),
		mcp.WithString("interval",
			mcp.Description("Candle interval: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 8h, 12h, 1d, 3d, 1w, 1M."),
			mcp.Required(),
		),
		mcp.WithNumber("start_time",
			mcp.Description("Start time in unix milliseconds (inclusive)."),
			mcp.Required(),
		),
		mcp.WithNumber("end_time",
			mcp.Description("End time in unix milliseconds (inclusive). Optional, defaults to now."),
		),
	)
}

func handleHyperliquidGetCandles(hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		coin, err := req.RequireString("coin")
		if err != nil {
			return mcp.NewToolResultError("coin is required"), nil
		}

		interval, err := req.RequireString("interval")
		if err != nil {
			return mcp.NewToolResultError("interval is required"), nil
		}
		validIntervals := map[string]bool{
			"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
			"1h": true, "2h": true, "4h": true, "8h": true, "12h": true,
			"1d": true, "3d": true, "1w": true, "1M": true,
		}
		if !validIntervals[interval] {
			return mcp.NewToolResultError("interval must be one of: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 8h, 12h, 1d, 3d, 1w, 1M"), nil
		}

		startTime := int64(req.GetFloat("start_time", 0))
		if startTime <= 0 {
			return mcp.NewToolResultError("start_time is required and must be positive"), nil
		}

		endTime := int64(req.GetFloat("end_time", 0))
		if endTime <= 0 {
			endTime = time.Now().UnixMilli()
		}
		if startTime > endTime {
			return mcp.NewToolResultError("start_time must not be after end_time"), nil
		}

		candles, err := hlClient.GetCandles(ctx, coin, interval, startTime, endTime)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get candles: %v", err)), nil
		}

		type candleOut struct {
			OpenTime  int64  `json:"open_time"`
			CloseTime int64  `json:"close_time"`
			Open      string `json:"open"`
			High      string `json:"high"`
			Low       string `json:"low"`
			Close     string `json:"close"`
			Volume    string `json:"volume"`
			Trades    int    `json:"trades"`
		}

		out := make([]candleOut, len(candles))
		for i, c := range candles {
			out[i] = candleOut{
				OpenTime:  c.OpenTime,
				CloseTime: c.CloseTime,
				Open:      c.Open,
				High:      c.High,
				Low:       c.Low,
				Close:     c.Close,
				Volume:    c.Volume,
				Trades:    c.Trades,
			}
		}

		result := map[string]any{
			"coin":     coin,
			"interval": interval,
			"candles":  out,
			"count":    len(out),
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
