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

func newHyperliquidBuildUsdTransferTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_usd_transfer",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid usdSend action to transfer USDC to another address on Hyperliquid L1. "+
				"This is an internal transfer within Hyperliquid, not a withdrawal to Arbitrum. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("destination",
			mcp.Description("Recipient address on Hyperliquid (0x-prefixed, 42-char)."),
			mcp.Required(),
		),
		mcp.WithString("amount",
			mcp.Description("USDC amount to send (e.g. \"100\")."),
			mcp.Required(),
		),
		mcp.WithString("address",
			mcp.Description("Sender's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildUsdTransfer(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		destination, err := req.RequireString("destination")
		if err != nil {
			return mcp.NewToolResultError("destination is required"), nil
		}

		amount, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("amount is required"), nil
		}

		nonce := time.Now().UnixMilli()

		actionPayload := map[string]any{
			"type":             "usdSend",
			"hyperliquidChain": "Mainnet",
			"signatureChainId": "0x66eee",
			"destination":      destination,
			"amount":           amount,
			"time":             nonce,
		}

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "usd_transfer",
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
