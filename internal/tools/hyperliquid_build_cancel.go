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

func newHyperliquidBuildCancelTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_cancel",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid cancel order action for the client to sign via EIP-712. "+
				"Cancels by order ID (oid) or client order ID (cloid). Provide exactly one. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("coin",
			mcp.Description("Coin symbol (e.g. BTC, ETH). Must match a name in the Hyperliquid universe."),
			mcp.Required(),
		),
		mcp.WithNumber("oid",
			mcp.Description("Order ID to cancel (from hyperliquid_get_open_orders). Provide oid or cloid."),
		),
		mcp.WithString("cloid",
			mcp.Description("Client order ID to cancel. Provide oid or cloid."),
		),
		mcp.WithBoolean("is_spot",
			mcp.Description("If true, cancel on spot market. Default: false (perpetual)."),
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

func handleHyperliquidBuildCancel(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		coin, err := req.RequireString("coin")
		if err != nil {
			return mcp.NewToolResultError("coin is required"), nil
		}

		oid := int64(req.GetFloat("oid", 0))
		cloid := req.GetString("cloid", "")
		isSpot := req.GetBool("is_spot", false)
		vaultAddress := req.GetString("vault_address", "")
		expiresAfter := int64(req.GetFloat("expires_after", 0))

		if oid == 0 && cloid == "" {
			return mcp.NewToolResultError("provide exactly one of oid or cloid"), nil
		}
		if oid != 0 && cloid != "" {
			return mcp.NewToolResultError("provide exactly one of oid or cloid, not both"), nil
		}

		var assetIndex int
		if isSpot {
			spotMeta, spotErr := hlClient.GetSpotMeta(ctx)
			if spotErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get spot meta: %v", spotErr)), nil
			}
			assetIndex, _, err = resolveSpotAssetIndex(spotMeta, coin)
		} else {
			perpMeta, perpErr := hlClient.GetPerpMeta(ctx)
			if perpErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get perp meta: %v", perpErr)), nil
			}
			assetIndex, _, err = resolveAssetIndex(perpMeta, coin)
		}
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var actionPayload map[string]any
		if cloid != "" {
			actionPayload = map[string]any{
				"type": "cancelByCloid",
				"cancels": []any{
					map[string]any{
						"a":     assetIndex,
						"cloid": cloid,
					},
				},
			}
		} else {
			actionPayload = map[string]any{
				"type": "cancel",
				"cancels": []any{
					map[string]any{
						"a": assetIndex,
						"o": oid,
					},
				},
			}
		}

		nonce := time.Now().UnixMilli()

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "cancel",
			"signing_mode":   "eip712",
			"address":        addr,
			"asset_index":    assetIndex,
			"coin":           coin,
			"nonce":          nonce,
			"action_payload": actionPayload,
			"exchange_url":   hlClient.ExchangeURL(),
		}

		if oid != 0 {
			result["oid"] = oid
		}
		if cloid != "" {
			result["cloid"] = cloid
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
