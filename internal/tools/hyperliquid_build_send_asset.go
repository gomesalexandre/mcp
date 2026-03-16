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

func newHyperliquidBuildSendAssetTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_send_asset",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid sendAsset action for generalized token transfers. "+
				"Supports transfers between addresses, DEXes, and subaccounts on Hyperliquid. "+
				"Only collateral tokens can transfer to/from perp DEXes. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("destination",
			mcp.Description("Recipient address (0x-prefixed, 42-char)."),
			mcp.Required(),
		),
		mcp.WithString("token",
			mcp.Description("Token identifier in format \"tokenName:tokenId\" (e.g. \"PURR:0xc4bf3f870c0e9465323c0b6ed28096c2\")."),
			mcp.Required(),
		),
		mcp.WithString("amount",
			mcp.Description("Quantity to transfer (e.g. \"0.01\")."),
			mcp.Required(),
		),
		mcp.WithString("source_dex",
			mcp.Description("Originating DEX name. Use \"\" for default USDC perp, \"spot\" for spot. Default: \"\"."),
		),
		mcp.WithString("destination_dex",
			mcp.Description("Target DEX name. Default: \"\"."),
		),
		mcp.WithString("from_sub_account",
			mcp.Description("Subaccount address (0x-prefixed) if transferring from a subaccount. Default: \"\"."),
		),
		mcp.WithString("address",
			mcp.Description("Sender's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildSendAsset(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
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

		token, err := req.RequireString("token")
		if err != nil {
			return mcp.NewToolResultError("token is required"), nil
		}

		amount, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("amount is required"), nil
		}

		sourceDex := req.GetString("source_dex", "")
		destinationDex := req.GetString("destination_dex", "")
		fromSubAccount := req.GetString("from_sub_account", "")

		nonce := time.Now().UnixMilli()

		actionPayload := map[string]any{
			"type":             "sendAsset",
			"hyperliquidChain": "Mainnet",
			"signatureChainId": "0xa4b1",
			"destination":      destination,
			"sourceDex":        sourceDex,
			"destinationDex":   destinationDex,
			"token":            token,
			"amount":           amount,
			"fromSubAccount":   fromSubAccount,
		}

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "send_asset",
			"signing_mode":   "eip712",
			"address":        addr,
			"destination":    destination,
			"token":          token,
			"amount":         amount,
			"nonce":          nonce,
			"action_payload": actionPayload,
			"exchange_url":   hlClient.ExchangeURL(),
		}

		if sourceDex != "" {
			result["source_dex"] = sourceDex
		}
		if destinationDex != "" {
			result["destination_dex"] = destinationDex
		}
		if fromSubAccount != "" {
			result["from_sub_account"] = fromSubAccount
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
