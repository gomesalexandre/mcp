package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	tonclient "github.com/vultisig/mcp/internal/ton"
	"github.com/vultisig/mcp/internal/vault"
)

func newGetTonBalanceTool() mcp.Tool {
	return mcp.NewTool("get_ton_balance",
		mcp.WithDescription(
			"Get the native TON balance for a TON address. "+
				"Returns balance in nanotons and TON. "+
				"The address parameter is required (TON address derivation is not available server-side).",
		),
		mcp.WithString("address",
			mcp.Description("TON address to check (from vault context addresses.Ton)"),
			mcp.Required(),
		),
	)
}

func handleGetTonBalance(store *vault.Store, tonClient *tonclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		address, err := req.RequireString("address")
		if err != nil {
			return mcp.NewToolResultError("missing address parameter (from vault context addresses.Ton)"), nil
		}
		if err := tonclient.ValidateAddress(address); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid TON address: %v", err)), nil
		}

		wallet, err := tonClient.GetWalletInfo(ctx, address)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to query TON balance: %v", err)), nil
		}

		balance, ok := new(big.Int).SetString(wallet.Balance, 10)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse wallet balance: %q", wallet.Balance)), nil
		}

		tonFloat := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			new(big.Float).SetFloat64(1e9),
		)

		result := map[string]any{
			"chain":          "Ton",
			"ticker":         "TON",
			"address":        address,
			"balance_nano":   wallet.Balance,
			"balance":        tonFloat.Text('f', 9),
			"status":         wallet.Status,
			"seqno":          wallet.Seqno,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
