package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

func newHyperliquidBuildModifyOrderTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_modify_order",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid modify order action for the client to sign via EIP-712. "+
				"Modifies a resting order in-place with new price and/or size. "+
				"Identify the order to modify by target_oid or target_cloid (exactly one required). "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithNumber("target_oid",
			mcp.Description("Order ID of the order to modify. Provide target_oid or target_cloid."),
		),
		mcp.WithString("target_cloid",
			mcp.Description("Client order ID of the order to modify. Provide target_oid or target_cloid."),
		),
		mcp.WithString("coin",
			mcp.Description("Coin symbol (e.g. BTC, ETH)."),
			mcp.Required(),
		),
		mcp.WithString("side",
			mcp.Description("Order side: buy or sell."),
			mcp.Required(),
		),
		mcp.WithString("size",
			mcp.Description("New order size in base units."),
			mcp.Required(),
		),
		mcp.WithString("price",
			mcp.Description("New limit price."),
			mcp.Required(),
		),
		mcp.WithString("tif",
			mcp.Description("Time-in-force: Gtc (default), Ioc, Alo."),
		),
		mcp.WithBoolean("reduce_only",
			mcp.Description("If true, order can only reduce an existing position. Default: false."),
		),
		mcp.WithBoolean("is_spot",
			mcp.Description("If true, modify on spot market. Default: false (perpetual)."),
		),
		mcp.WithString("cloid",
			mcp.Description("New client order ID for the modified order. Optional."),
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

func handleHyperliquidBuildModifyOrder(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		targetOid := int64(req.GetFloat("target_oid", 0))
		targetCloid := req.GetString("target_cloid", "")

		if targetOid == 0 && targetCloid == "" {
			return mcp.NewToolResultError("provide exactly one of target_oid or target_cloid"), nil
		}
		if targetOid != 0 && targetCloid != "" {
			return mcp.NewToolResultError("provide exactly one of target_oid or target_cloid, not both"), nil
		}

		coin, err := req.RequireString("coin")
		if err != nil {
			return mcp.NewToolResultError("coin is required"), nil
		}

		side, err := req.RequireString("side")
		if err != nil {
			return mcp.NewToolResultError("side is required"), nil
		}
		side = strings.ToLower(side)
		if side != "buy" && side != "sell" {
			return mcp.NewToolResultError("side must be 'buy' or 'sell'"), nil
		}

		sizeStr, err := req.RequireString("size")
		if err != nil {
			return mcp.NewToolResultError("size is required"), nil
		}

		priceStr, err := req.RequireString("price")
		if err != nil {
			return mcp.NewToolResultError("price is required"), nil
		}

		tif := req.GetString("tif", "Gtc")
		reduceOnly := req.GetBool("reduce_only", false)
		isSpot := req.GetBool("is_spot", false)
		cloid := req.GetString("cloid", "")
		vaultAddress := req.GetString("vault_address", "")
		expiresAfter := int64(req.GetFloat("expires_after", 0))

		var assetIndex int
		var szDecimals int
		if isSpot {
			spotMeta, spotErr := hlClient.GetSpotMeta(ctx)
			if spotErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get spot meta: %v", spotErr)), nil
			}
			assetIndex, szDecimals, err = resolveSpotAssetIndex(spotMeta, coin)
		} else {
			perpMeta, perpErr := hlClient.GetPerpMeta(ctx)
			if perpErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to get perp meta: %v", perpErr)), nil
			}
			assetIndex, szDecimals, err = resolveAssetIndex(perpMeta, coin)
		}
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		truncatedSize, err := truncateToDecimals(sizeStr, szDecimals)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid size: %v", err)), nil
		}

		normalizedPrice, err := normalizePrice(priceStr, szDecimals, isSpot)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid price: %v", err)), nil
		}

		orderObj := map[string]any{
			"a": assetIndex,
			"b": side == "buy",
			"p": normalizedPrice,
			"s": truncatedSize,
			"r": reduceOnly,
			"t": map[string]any{
				"limit": map[string]any{
					"tif": tif,
				},
			},
		}
		if cloid != "" {
			orderObj["c"] = cloid
		}

		var oidValue any
		if targetOid != 0 {
			oidValue = targetOid
		} else {
			oidValue = targetCloid
		}

		actionPayload := map[string]any{
			"type":  "modify",
			"oid":   oidValue,
			"order": orderObj,
		}

		nonce := time.Now().UnixMilli()

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "modify",
			"signing_mode":   "eip712",
			"address":        addr,
			"asset_index":    assetIndex,
			"coin":           coin,
			"side":           side,
			"size":           truncatedSize,
			"price":          normalizedPrice,
			"nonce":          nonce,
			"action_payload": actionPayload,
			"exchange_url":   hlClient.ExchangeURL(),
		}

		if targetOid != 0 {
			result["target_oid"] = targetOid
		}
		if targetCloid != "" {
			result["target_cloid"] = targetCloid
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
