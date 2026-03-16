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

func newHyperliquidGetPerpStateTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_perp_state",
		mcp.WithDescription(
			"Query perpetual positions, margin summary, and account value on Hyperliquid. "+
				"Returns open positions with entry price, liquidation price, PnL, and leverage. "+
				"Positions with zero size are filtered out. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("address",
			mcp.Description("Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidGetPerpState(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")

		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		state, err := hlClient.GetPerpState(ctx, addr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get perp state: %v", err)), nil
		}

		var positions []hyperliquid.AssetPosition
		for _, ap := range state.AssetPositions {
			if ap.Position.Szi != "0" && ap.Position.Szi != "0.0" && ap.Position.Szi != "" {
				positions = append(positions, ap)
			}
		}

		result := map[string]any{
			"address":              addr,
			"margin_summary":       state.MarginSummary,
			"cross_margin_summary": state.CrossMarginSummary,
			"withdrawable":         state.Withdrawable,
			"positions":            positions,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
