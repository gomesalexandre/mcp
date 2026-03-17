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

func newHyperliquidBuildWithdrawTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_withdraw",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid withdraw action to move USDC from Hyperliquid to Arbitrum. "+
				"Withdrawals take approximately 5 minutes to finalize and incur a ~$1 fee. "+
				"If destination is omitted, the user's own address is used. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("amount",
			mcp.Description("USDC amount to withdraw (e.g. \"100\")."),
			mcp.Required(),
		),
		mcp.WithString("destination",
			mcp.Description("Arbitrum destination address (0x-prefixed, 42-char). Defaults to the user's own address."),
		),
		mcp.WithString("address",
			mcp.Description("User's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildWithdraw(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
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

		destination := req.GetString("destination", addr)

		nonce := time.Now().UnixMilli()

		actionPayload := map[string]any{
			"type":             "withdraw3",
			"hyperliquidChain": "Mainnet",
			"signatureChainId": "0x66eee",
			"amount":           amount,
			"time":             nonce,
			"destination":      destination,
		}

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "withdraw",
			"signing_mode":   "eip712",
			"address":        addr,
			"destination":    destination,
			"amount":         amount,
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
