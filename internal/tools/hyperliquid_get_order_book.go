package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
)

func newHyperliquidGetOrderBookTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_order_book",
		mcp.WithDescription(
			"Query the L2 order book for a specific coin on Hyperliquid. "+
				"Returns bid and ask levels with prices, sizes, and order counts. "+
				"For perpetuals use the coin symbol (e.g. BTC, ETH). "+
				"For spot tokens use @{index} from spotMeta or the canonical pair name (e.g. PURR/USDC).",
		),
		mcp.WithString("coin",
			mcp.Description("Coin identifier. Perps: symbol (BTC). Spot: @{index} or pair name (PURR/USDC)."),
			mcp.Required(),
		),
		mcp.WithNumber("n_sig_figs",
			mcp.Description("Number of significant figures for price level aggregation (2-5). Default: 5."),
		),
	)
}

func handleHyperliquidGetOrderBook(hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		coin, err := req.RequireString("coin")
		if err != nil {
			return mcp.NewToolResultError("coin is required"), nil
		}

		nSigFigs := int(req.GetFloat("n_sig_figs", 5))
		if nSigFigs < 2 || nSigFigs > 5 {
			return mcp.NewToolResultError("n_sig_figs must be between 2 and 5"), nil
		}

		book, err := hlClient.GetL2Book(ctx, coin, nSigFigs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get order book: %v", err)), nil
		}

		result := map[string]any{
			"coin": book.Coin,
			"time": book.Time,
		}
		if len(book.Levels) > 0 {
			result["bids"] = book.Levels[0]
		}
		if len(book.Levels) > 1 {
			result["asks"] = book.Levels[1]
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
