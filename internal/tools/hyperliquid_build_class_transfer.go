package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

func newHyperliquidBuildClassTransferTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_class_transfer",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid usdClassTransfer action to move USDC between spot and perp margin. "+
				"Set to_perp=true to move from spot to perp, or to_perp=false to move from perp to spot. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("amount",
			mcp.Description("USDC amount to transfer (e.g. \"100\")."),
			mcp.Required(),
		),
		mcp.WithBoolean("to_perp",
			mcp.Description("Direction: true = spot → perp, false = perp → spot."),
			mcp.Required(),
		),
		mcp.WithString("address",
			mcp.Description("User's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildClassTransfer(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		amount, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("amount is required"), nil
		}
		amountVal, _, err := big.ParseFloat(amount, 10, 128, big.ToNearestEven)
		if err != nil || amountVal.Sign() <= 0 {
			return mcp.NewToolResultError("amount must be a positive number"), nil
		}

		toPerp := req.GetBool("to_perp", true)

		nonce := time.Now().UnixMilli()

		actionPayload := map[string]any{
			"type":             "usdClassTransfer",
			"hyperliquidChain": "Mainnet",
			"signatureChainId": "0x66eee",
			"amount":           amount,
			"toPerp":           toPerp,
			"nonce":            nonce,
		}

		direction := "spot_to_perp"
		if !toPerp {
			direction = "perp_to_spot"
		}

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "class_transfer",
			"signing_mode":   "eip712",
			"address":        addr,
			"amount":         amount,
			"direction":      direction,
			"to_perp":        toPerp,
			"nonce":          nonce,
			"action_payload": actionPayload,
			"exchange_url":   hlClient.ExchangeURL(),
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
