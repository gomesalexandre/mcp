package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	tonclient "github.com/vultisig/mcp/internal/ton"
)

func newBuildTonSendTool() mcp.Tool {
	return mcp.NewTool("build_ton_send",
		mcp.WithDescription(
			"Prepare parameters for a TON (The Open Network) native transfer transaction. "+
				"Returns all required fields (from, to, amount, seqno, chain) for the client to build and sign the transaction. "+
				"Amount is in nanotons (1 TON = 1,000,000,000 nanotons). "+
				"The 'from' address must be provided (the client app sends it in the context addresses).",
		),
		mcp.WithString("to",
			mcp.Description("Recipient TON address (user-friendly format, EQ... or UQ...)"),
			mcp.Required(),
		),
		mcp.WithString("amount",
			mcp.Description("Amount in nanotons (1 TON = 1,000,000,000 nanotons, decimal string)"),
			mcp.Required(),
		),
		mcp.WithString("from",
			mcp.Description("Sender TON address (from vault context addresses)"),
			mcp.Required(),
		),
		mcp.WithString("memo",
			mcp.Description("Optional memo/comment for the transfer"),
		),
	)
}

func handleBuildTonSend(tonClient *tonclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		toAddr, err := req.RequireString("to")
		if err != nil {
			return mcp.NewToolResultError("missing to parameter"), nil
		}
		if err := tonclient.ValidateAddress(toAddr); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid recipient address: %v", err)), nil
		}

		amountStr, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("missing amount parameter"), nil
		}
		amountNano, ok := new(big.Int).SetString(amountStr, 10)
		if !ok || amountNano.Sign() <= 0 {
			return mcp.NewToolResultError(fmt.Sprintf("invalid amount: %q (must be a positive integer in nanotons)", amountStr)), nil
		}

		fromAddr, err := req.RequireString("from")
		if err != nil {
			return mcp.NewToolResultError("missing from parameter (sender TON address from vault context)"), nil
		}
		if err := tonclient.ValidateAddress(fromAddr); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid sender address: %v", err)), nil
		}

		memo := req.GetString("memo", "")

		// Get wallet info (balance + seqno)
		wallet, err := tonClient.GetWalletInfo(ctx, fromAddr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get wallet info: %v", err)), nil
		}

		// Check balance
		balance, ok := new(big.Int).SetString(wallet.Balance, 10)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse wallet balance: %q", wallet.Balance)), nil
		}
		if balance.Cmp(amountNano) < 0 {
			return mcp.NewToolResultError(fmt.Sprintf(
				"insufficient balance: have %s nanotons (%s TON), need %s nanotons",
				wallet.Balance, formatNanotons(balance), amountStr,
			)), nil
		}

		result := map[string]any{
			"chain":   "Ton",
			"action":  "transfer",
			"from":    fromAddr,
			"to":      toAddr,
			"amount":  amountStr,
			"ticker":  "TON",
			"seqno":   wallet.Seqno,
		}

		if memo != "" {
			result["memo"] = memo
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func formatNanotons(nanotons *big.Int) string {
	tonFloat := new(big.Float).Quo(
		new(big.Float).SetInt(nanotons),
		new(big.Float).SetFloat64(1e9),
	)
	return tonFloat.Text('f', 4)
}
