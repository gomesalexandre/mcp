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

func newHyperliquidGetUserRateLimitTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_user_rate_limit",
		mcp.WithDescription(
			"Query API rate limit status on Hyperliquid for a user address. "+
				"Returns cumulative volume, requests used, request cap, and surplus. "+
				"Useful for monitoring rate limit consumption before heavy queries. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("address",
			mcp.Description("Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidGetUserRateLimit(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")

		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		rateLimit, err := hlClient.GetUserRateLimit(ctx, addr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get user rate limit: %v", err)), nil
		}

		result := map[string]any{
			"address":            addr,
			"cum_vlm":            rateLimit.CumVlm,
			"n_requests_used":    rateLimit.NRequestsUsed,
			"n_requests_cap":     rateLimit.NRequestsCap,
			"n_requests_surplus": rateLimit.NRequestsSurplus,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
