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

func newHyperliquidBuildUnstakeTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_unstake",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid cWithdraw action to unstake HYPE tokens. "+
				"Unstaked tokens undergo a 7-day unstaking queue before reaching the spot account. "+
				"The wei parameter is the amount of HYPE in wei (1 HYPE = 1e18 wei). "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("wei",
			mcp.Description("Amount of HYPE in wei to unstake (integer string, e.g. \"1000000000000000000\" for 1 HYPE)."),
			mcp.Required(),
		),
		mcp.WithString("address",
			mcp.Description("User's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildUnstake(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		weiStr, err := req.RequireString("wei")
		if err != nil {
			return mcp.NewToolResultError("wei is required"), nil
		}

		weiBig, ok := new(big.Int).SetString(weiStr, 10)
		if !ok || weiBig.Sign() <= 0 {
			return mcp.NewToolResultError("wei must be a positive integer"), nil
		}

		nonce := time.Now().UnixMilli()

		weiNum, _ := weiBig.Float64()

		actionPayload := map[string]any{
			"type":             "cWithdraw",
			"hyperliquidChain": "Mainnet",
			"signatureChainId": "0xa4b1",
			"wei":              weiNum,
			"nonce":            nonce,
		}

		result := map[string]any{
			"chain":          "Hyperliquid",
			"action":         "unstake",
			"signing_mode":   "eip712",
			"address":        addr,
			"wei":            weiStr,
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
