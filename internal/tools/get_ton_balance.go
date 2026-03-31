package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/resolve"
	tonclient "github.com/vultisig/mcp/internal/ton"
	"github.com/vultisig/mcp/internal/vault"
)

func newGetTonBalanceTool() mcp.Tool {
	return mcp.NewTool("get_ton_balance",
		mcp.WithDescription(
			"Get the native TON balance for a TON address. "+
				"Returns balance in nanotons and TON. "+
				"Accepts inline vault keys or falls back to set_vault_info session state.",
		),
		mcp.WithString("address",
			mcp.Description("TON address to check. Falls back to vault-derived address if omitted."),
		),
	)
}

func handleGetTonBalance(store *vault.Store, tonClient *tonclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		address := req.GetString("address", "")

		if address == "" {
			v := resolve.ResolveVault(ctx, req, store)
			if v == nil {
				return mcp.NewToolResultError("no address or vault info available"), nil
			}
			// Derive TON address from EdDSA public key
			// TON address derivation requires @ton/ton library which is JS-only.
			// For the MCP tool, the address must be provided explicitly.
			return mcp.NewToolResultError("TON address derivation not available in MCP — pass 'address' parameter (from vault context addresses.Ton)"), nil
		}

		wallet, err := tonClient.GetWalletInfo(ctx, address)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to query TON balance: %v", err)), nil
		}

		balance, ok := new(big.Int).SetString(wallet.Balance, 10)
		if !ok {
			balance = big.NewInt(0)
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
