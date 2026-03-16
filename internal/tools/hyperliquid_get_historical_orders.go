package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

func newHyperliquidGetHistoricalOrdersTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_historical_orders",
		mcp.WithDescription(
			"Query up to 2000 recent historical orders on Hyperliquid. "+
				"Returns past orders with their status (filled, canceled, triggered, rejected, marginCanceled). "+
				"WARNING: This endpoint has extra rate-limit weight proportional to returned items. Use sparingly in agentic loops. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("address",
			mcp.Description("Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidGetHistoricalOrders(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")

		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		orders, err := hlClient.GetHistoricalOrders(ctx, addr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get historical orders: %v", err)), nil
		}

		result := map[string]any{
			"address": addr,
			"orders":  orders,
			"count":   len(orders),
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
