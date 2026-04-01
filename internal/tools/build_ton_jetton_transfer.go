package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	tonclient "github.com/vultisig/mcp/internal/ton"
)

func newBuildTonJettonTransferTool() mcp.Tool {
	return mcp.NewTool("build_ton_jetton_transfer",
		mcp.WithDescription(
			"Prepare parameters for a TON jetton (token) transfer. "+
				"Resolves the sender's jetton wallet address and verifies sufficient balance. "+
				"Amount is in base units (e.g. 1000000 for 1 USDT with 6 decimals). "+
				"The 'from' address is the owner address (from vault context addresses), not the jetton wallet.",
		),
		mcp.WithString("from",
			mcp.Description("Sender TON address - the owner of the jetton wallet (from vault context addresses)"),
			mcp.Required(),
		),
		mcp.WithString("to",
			mcp.Description("Recipient TON address (user-friendly format, EQ... or UQ...)"),
			mcp.Required(),
		),
		mcp.WithString("jetton_master",
			mcp.Description("Jetton master contract address (e.g. EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs for USDT)"),
			mcp.Required(),
		),
		mcp.WithString("amount",
			mcp.Description("Amount in base units (decimal string, e.g. \"1000000\" for 1 USDT)"),
			mcp.Required(),
		),
		mcp.WithNumber("decimals",
			mcp.Description("Token decimals (e.g. 6 for USDT, 9 for native TON). Required for correct amount display."),
			mcp.Required(),
		),
	)
}

func handleBuildTonJettonTransfer(tonClient *tonclient.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fmt.Fprintf(os.Stderr, "[mcp] [CALL] build_ton_jetton_transfer\n")
		fromAddr, err := req.RequireString("from")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: missing from parameter\n")
			return mcp.NewToolResultError("missing from parameter (sender TON address from vault context)"), nil
		}
		if err := tonclient.ValidateAddress(fromAddr); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: invalid sender address: %v\n", err)
			return mcp.NewToolResultError(fmt.Sprintf("invalid sender address: %v", err)), nil
		}

		toAddr, err := req.RequireString("to")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: missing to parameter\n")
			return mcp.NewToolResultError("missing to parameter"), nil
		}
		if err := tonclient.ValidateAddress(toAddr); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: invalid recipient address: %v\n", err)
			return mcp.NewToolResultError(fmt.Sprintf("invalid recipient address: %v", err)), nil
		}

		jettonMaster, err := req.RequireString("jetton_master")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: missing jetton_master parameter\n")
			return mcp.NewToolResultError("missing jetton_master parameter"), nil
		}
		if err := tonclient.ValidateAddress(jettonMaster); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: invalid jetton master address: %v\n", err)
			return mcp.NewToolResultError(fmt.Sprintf("invalid jetton master address: %v", err)), nil
		}

		amountStr, err := req.RequireString("amount")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: missing amount parameter\n")
			return mcp.NewToolResultError("missing amount parameter"), nil
		}
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok || amount.Sign() <= 0 {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: invalid amount %q\n", amountStr)
			return mcp.NewToolResultError(fmt.Sprintf("invalid amount: %q (must be a positive integer in base units)", amountStr)), nil
		}

		// Resolve the sender's jetton wallet address and balance
		jettonWallet, err := tonClient.GetJettonWallet(ctx, fromAddr, jettonMaster)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [RPC_ERR] build_ton_jetton_transfer GetJettonWallet from=%s jetton=%s: %v\n", fromAddr, jettonMaster, err)
			return mcp.NewToolResultError(fmt.Sprintf("failed to get jetton wallet: %v", err)), nil
		}

		if jettonWallet.Address == "" {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: no jetton wallet from=%s jetton=%s\n", fromAddr, jettonMaster)
			return mcp.NewToolResultError(fmt.Sprintf(
				"no jetton wallet found for owner %s and jetton master %s - sender may not hold this token",
				fromAddr, jettonMaster,
			)), nil
		}

		// Check balance
		balance, ok := new(big.Int).SetString(jettonWallet.Balance, 10)
		if !ok {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: failed to parse balance %q\n", jettonWallet.Balance)
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse jetton balance: %q", jettonWallet.Balance)), nil
		}
		if balance.Cmp(amount) < 0 {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: insufficient balance have=%s need=%s jetton=%s\n", jettonWallet.Balance, amountStr, jettonMaster)
			return mcp.NewToolResultError(fmt.Sprintf(
				"insufficient jetton balance: have %s, need %s (jetton master: %s)",
				jettonWallet.Balance, amountStr, jettonMaster,
			)), nil
		}

		// Read decimals from request - required, validated
		decimalsFloat, err := req.RequireFloat("decimals")
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: missing decimals parameter\n")
			return mcp.NewToolResultError("missing decimals parameter"), nil
		}
		decimals := int(decimalsFloat)
		if decimals < 0 || decimals > 18 {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: invalid decimals %d\n", decimals)
			return mcp.NewToolResultError(fmt.Sprintf("invalid decimals: %d (must be 0-18)", decimals)), nil
		}
		humanAmount := formatJettonBaseUnits(amount, decimals)

		result := map[string]any{
			"chain":         "Ton",
			"action":        "jetton_transfer",
			"from":          fromAddr,
			"to":            toAddr,
			"jetton_master": jettonMaster,
			"jetton_wallet": jettonWallet.Address,
			"amount":        humanAmount,
			"amount_base":   amountStr,
			"decimals":      decimals,
		}

		data, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[mcp] [FAIL] build_ton_jetton_transfer: marshal error: %v\n", err)
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		fmt.Fprintf(os.Stderr, "[mcp] [OK] build_ton_jetton_transfer: %s %s -> %s (jetton: %s)\n", humanAmount, jettonMaster, toAddr, jettonWallet.Address)
		return mcp.NewToolResultText(string(data)), nil
	}
}

func formatJettonBaseUnits(amount *big.Int, decimals int) string {
	if decimals <= 0 {
		return amount.String()
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	result := new(big.Float).Quo(
		new(big.Float).SetInt(amount),
		new(big.Float).SetInt(divisor),
	)
	return result.Text('f', decimals)
}
