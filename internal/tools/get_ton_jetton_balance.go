package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	tonclient "github.com/vultisig/mcp/internal/ton"
)

func newGetTonJettonBalanceTool() mcp.Tool {
	return mcp.NewTool("get_ton_jetton_balance",
		mcp.WithDescription(
			"Get balance of a specific TON jetton token. "+
				"Returns balance in base units and the jetton wallet address.",
		),
		mcp.WithString("address",
			mcp.Description("TON owner address (EQ.../UQ.../raw format)"),
			mcp.Required(),
		),
		mcp.WithString("jetton_master",
			mcp.Description("Jetton master contract address"),
			mcp.Required(),
		),
	)
}

func handleGetTonJettonBalance(tonClient *tonclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		address, err := req.RequireString("address")
		if err != nil {
			return mcp.NewToolResultError("missing address parameter"), nil
		}
		if err := tonclient.ValidateAddress(address); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid TON address: %v", err)), nil
		}

		jettonMaster, err := req.RequireString("jetton_master")
		if err != nil {
			return mcp.NewToolResultError("missing jetton_master parameter"), nil
		}

		wallet, err := tonClient.GetJettonWallet(ctx, address, jettonMaster)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get jetton balance: %v", err)), nil
		}

		result := map[string]any{
			"jetton_master":  jettonMaster,
			"balance":        wallet.Balance,
			"wallet_address": wallet.Address,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
