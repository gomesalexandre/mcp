package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

func newHyperliquidBuildScheduleCancelTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_schedule_cancel",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid scheduleCancel action (dead-man switch). "+
				"When triggered, cancels all open orders. Max 10 triggers per day (resets 00:00 UTC). "+
				"Set time to a future unix timestamp (>= 5 seconds ahead) to schedule. "+
				"Omit time to clear a previously scheduled cancel. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithNumber("time",
			mcp.Description("Unix timestamp in milliseconds for when to cancel all orders. Must be >= 5 seconds in the future. Omit to clear."),
		),
		mcp.WithString("vault_address",
			mcp.Description("Vault or subaccount address for delegated trading. Optional."),
		),
		mcp.WithNumber("expires_after",
			mcp.Description("Unix timestamp in milliseconds after which the signed action expires."),
		),
		mcp.WithString("address",
			mcp.Description("User's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildScheduleCancel(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		cancelTime := int64(req.GetFloat("time", 0))
		vaultAddress := req.GetString("vault_address", "")
		expiresAfter := int64(req.GetFloat("expires_after", 0))

		actionPayload := map[string]any{
			"type": "scheduleCancel",
		}

		actionDesc := "clear_schedule_cancel"
		if cancelTime > 0 {
			nowMs := time.Now().UnixMilli()
			if cancelTime < nowMs+5000 {
				return mcp.NewToolResultError("time must be at least 5 seconds in the future"), nil
			}
			actionPayload["time"] = cancelTime
			actionDesc = "schedule_cancel"
		}

		nonce := time.Now().UnixMilli()

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         actionDesc,
			"signing_mode":   "eip712",
			"address":        addr,
			"nonce":          nonce,
			"action_payload": actionPayload,
			"exchange_url":   hlClient.ExchangeURL(),
		}

		if cancelTime > 0 {
			result["cancel_time"] = cancelTime
		}
		if vaultAddress != "" {
			result["vault_address"] = vaultAddress
		}
		if expiresAfter > 0 {
			result["expires_after"] = expiresAfter
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
