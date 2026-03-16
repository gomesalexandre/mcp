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

func newHyperliquidBuildOrderTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_order",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid order action for the client to sign via EIP-712. "+
				"Returns the order action payload, asset index, and metadata. "+
				"The client must sign the typed data and submit to /exchange. "+
				"Supports limit (Gtc, Ioc, Alo) and market orders for perpetuals and spot. "+
				"Market orders are encoded as aggressive IOC limits at the user-supplied worst-case price. "+
				"Size is truncated and price is normalized to comply with Hyperliquid precision rules. "+
				"Load the 'hyperliquid-trading' skill for required pre-checks and confirmation flow. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("coin",
			mcp.Description("Coin symbol (e.g. BTC, ETH). Must match a name in the Hyperliquid universe."),
			mcp.Required(),
		),
		mcp.WithString("side",
			mcp.Description("Order side: buy or sell."),
			mcp.Required(),
		),
		mcp.WithString("size",
			mcp.Description("Order size in base units (e.g. 0.01 for 0.01 BTC). Will be truncated to szDecimals."),
			mcp.Required(),
		),
		mcp.WithString("price",
			mcp.Description("Limit price. For market orders: worst acceptable price. Normalized to Hyperliquid tick rules."),
			mcp.Required(),
		),
		mcp.WithString("order_type",
			mcp.Description("Order type: limit (default) or market. Market is encoded as IOC at the given price."),
		),
		mcp.WithString("tif",
			mcp.Description("Time-in-force for limit orders: Gtc (default), Ioc, Alo."),
		),
		mcp.WithBoolean("reduce_only",
			mcp.Description("If true, order can only reduce an existing position. Default: false."),
		),
		mcp.WithBoolean("is_spot",
			mcp.Description("If true, trade on spot market. Default: false (perpetual)."),
		),
		mcp.WithString("cloid",
			mcp.Description("Client order ID for reconciliation. Max 128 chars, alphanumeric/dash/underscore."),
		),
		mcp.WithString("vault_address",
			mcp.Description("Vault or subaccount address for delegated trading. Optional."),
		),
		mcp.WithNumber("expires_after",
			mcp.Description("Unix timestamp in milliseconds after which the signed action expires. Prevents stale payloads."),
		),
		mcp.WithString("address",
			mcp.Description("User's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildOrder(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
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

		orderType := req.GetString("order_type", "limit")
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

		if orderType == "market" {
			tif = "Ioc"
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

		actionPayload := map[string]any{
			"type":     "order",
			"orders":   []any{orderObj},
			"grouping": "na",
		}

		nonce := time.Now().UnixMilli()

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "order",
			"signing_mode":   "eip712",
			"address":        addr,
			"asset_index":    assetIndex,
			"coin":           coin,
			"side":           side,
			"size":           truncatedSize,
			"price":          normalizedPrice,
			"order_type":     orderType,
			"tif":            tif,
			"reduce_only":    reduceOnly,
			"nonce":          nonce,
			"action_payload": actionPayload,
			"exchange_url":   hlClient.ExchangeURL(),
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
