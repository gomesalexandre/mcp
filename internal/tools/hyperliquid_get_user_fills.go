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

func newHyperliquidGetUserFillsTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_user_fills",
		mcp.WithDescription(
			"Query trade fill history on Hyperliquid for a user address. "+
				"Returns up to 2000 fills with coin, side, price, size, fee, realized PnL, and direction. "+
				"When start_time is provided, uses time-ranged query (userFillsByTime). "+
				"Rate-limit weight scales with returned item count — prefer narrow time ranges in agentic loops. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("address",
			mcp.Description("Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
		mcp.WithNumber("start_time",
			mcp.Description("Start time in unix milliseconds (inclusive). If provided, uses time-ranged query."),
		),
		mcp.WithNumber("end_time",
			mcp.Description("End time in unix milliseconds (inclusive). Only used when start_time is provided."),
		),
	)
}

func handleHyperliquidGetUserFills(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")

		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		startTime := int64(req.GetFloat("start_time", 0))
		endTime := int64(req.GetFloat("end_time", 0))

		fills, err := hlClient.GetUserFills(ctx, addr, startTime, endTime)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get user fills: %v", err)), nil
		}

		result := map[string]any{
			"address": addr,
			"fills":   fills,
			"count":   len(fills),
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
