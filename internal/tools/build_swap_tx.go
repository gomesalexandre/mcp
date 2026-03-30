package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"

	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/recipes/sdk/swap"
)

// lifiChainIDs maps chain names to LiFi chain IDs (same as recipes SDK).
var lifiChainIDs = map[string]int{
	"Solana": 1151111081099710,
}

// resolveSolanaTokenAddress looks up the canonical token address from LiFi's /tokens endpoint.
// Returns the canonical address if found, or the original if lookup fails.
func resolveSolanaTokenAddress(symbol, address string, chainID int) string {
	if address == "" || symbol == "" {
		return address
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://li.quest/v1/token?chain=%d&token=%s", chainID, address))
	if err != nil || resp.StatusCode != 200 {
		// Token not found with this address - try by symbol via search
		if resp != nil {
			resp.Body.Close()
		}
		// Fallback: search all tokens for this chain and match by symbol
		resp2, err2 := client.Get(fmt.Sprintf("https://li.quest/v1/tokens?chains=%d", chainID))
		if err2 != nil || resp2.StatusCode != 200 {
			if resp2 != nil {
				resp2.Body.Close()
			}
			return address
		}
		defer resp2.Body.Close()
		body, _ := io.ReadAll(resp2.Body)
		var tokensResp struct {
			Tokens map[string][]struct {
				Address string `json:"address"`
				Symbol  string `json:"symbol"`
			} `json:"tokens"`
		}
		if json.Unmarshal(body, &tokensResp) != nil {
			return address
		}
		chainKey := fmt.Sprintf("%d", chainID)
		for _, t := range tokensResp.Tokens[chainKey] {
			if strings.EqualFold(t.Symbol, symbol) && t.Address != "" {
				return t.Address
			}
		}
		return address
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tokenResp struct {
		Address string `json:"address"`
	}
	if json.Unmarshal(body, &tokenResp) == nil && tokenResp.Address != "" {
		return tokenResp.Address
	}
	return address
}


func newBuildSwapTxTool() mcp.Tool {
	return mcp.NewTool("build_swap_tx",
		mcp.WithDescription("Build unsigned transaction(s) for a token swap. Supports same-chain and cross-chain swaps across EVM, Solana, and other chains via THORChain, Mayachain, 1inch, LiFi, Jupiter, and Uniswap. Response includes chain field and swap parameters for clients that prefer local transaction building. EVM: swap_tx.data is hex calldata with 0x prefix. Solana: swap_tx.data is base64-encoded serialized transaction. Load the 'swap-trading' skill for required pre-checks and confirmation flow."),
		mcp.WithString("from_chain", mcp.Description("Source chain (e.g. \"Ethereum\", \"Bitcoin\", \"Solana\")"), mcp.Required()),
		mcp.WithString("from_symbol", mcp.Description("Source token symbol (e.g. \"ETH\", \"USDC\")"), mcp.Required()),
		mcp.WithString("from_address", mcp.Description("Source token contract address (empty for native coins)")),
		mcp.WithNumber("from_decimals", mcp.Description("Source token decimals (e.g. 18 for ETH, 6 for USDC)"), mcp.Required()),
		mcp.WithString("to_chain", mcp.Description("Destination chain"), mcp.Required()),
		mcp.WithString("to_symbol", mcp.Description("Destination token symbol"), mcp.Required()),
		mcp.WithString("to_address", mcp.Description("Destination token contract address (empty for native coins)")),
		mcp.WithNumber("to_decimals", mcp.Description("Destination token decimals"), mcp.Required()),
		mcp.WithString("amount", mcp.Description("Amount in base units (e.g. \"1000000\" for 1 USDC)"), mcp.Required()),
		mcp.WithString("sender", mcp.Description("Sender wallet address"), mcp.Required()),
		mcp.WithString("destination", mcp.Description("Destination wallet address"), mcp.Required()),
	)
}

type swapResult struct {
	Chain          string          `json:"chain"`
	Provider       string          `json:"provider"`
	ExpectedOutput string          `json:"expected_output"`
	MinimumOutput  string          `json:"minimum_output"`
	NeedsApproval  bool            `json:"needs_approval"`
	ApprovalTx     json.RawMessage `json:"approval_tx,omitempty"`
	SwapTx         json.RawMessage `json:"swap_tx"`
	Memo           string          `json:"memo,omitempty"`
	// Swap parameters echoed back for clients that prefer local building.
	FromChain    string `json:"from_chain"`
	FromSymbol   string `json:"from_symbol"`
	FromAddress  string `json:"from_address,omitempty"`
	FromDecimals int    `json:"from_decimals"`
	ToChain      string `json:"to_chain"`
	ToSymbol     string `json:"to_symbol"`
	ToAddress    string `json:"to_address,omitempty"`
	ToDecimals   int    `json:"to_decimals"`
	Amount       string `json:"amount"`
	Sender       string `json:"sender"`
	Destination  string `json:"destination"`
}

type swapTxJSON struct {
	To       string `json:"to"`
	Value    string `json:"value"`
	Data     string `json:"data,omitempty"`
	Memo     string `json:"memo,omitempty"`
	GasLimit uint64 `json:"gas_limit,omitempty"`
}

func handleBuildSwapTx(svc *swap.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fromChain, err := req.RequireString("from_chain")
		if err != nil {
			return mcp.NewToolResultError("missing from_chain"), nil
		}
		fromSymbol, err := req.RequireString("from_symbol")
		if err != nil {
			return mcp.NewToolResultError("missing from_symbol"), nil
		}
		fromAddress := req.GetString("from_address", "")
		fromDecimals := int(req.GetInt("from_decimals", 18))

		toChain, err := req.RequireString("to_chain")
		if err != nil {
			return mcp.NewToolResultError("missing to_chain"), nil
		}
		toSymbol, err := req.RequireString("to_symbol")
		if err != nil {
			return mcp.NewToolResultError("missing to_symbol"), nil
		}
		toAddress := req.GetString("to_address", "")
		toDecimals := int(req.GetInt("to_decimals", 18))

		// For Solana: resolve canonical token addresses via LiFi to handle AI case-mangling.
		// Solana base58 addresses are case-sensitive and lowercasing produces a different (invalid) address.
		if toChain == "Solana" && toAddress != "" {
			if chainID, ok := lifiChainIDs[toChain]; ok {
				toAddress = resolveSolanaTokenAddress(toSymbol, toAddress, chainID)
			}
		}
		if fromChain == "Solana" && fromAddress != "" {
			if chainID, ok := lifiChainIDs[fromChain]; ok {
				fromAddress = resolveSolanaTokenAddress(fromSymbol, fromAddress, chainID)
			}
		}

		amountStr, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("missing amount"), nil
		}
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("invalid amount: %q", amountStr)), nil
		}

		sender, err := req.RequireString("sender")
		if err != nil {
			return mcp.NewToolResultError("missing sender"), nil
		}
		destination, err := req.RequireString("destination")
		if err != nil {
			return mcp.NewToolResultError("missing destination"), nil
		}
		params := swap.SwapParams{
			FromChain:    fromChain,
			FromSymbol:   fromSymbol,
			FromAddress:  fromAddress,
			FromDecimals: fromDecimals,
			ToChain:      toChain,
			ToSymbol:     toSymbol,
			ToAddress:    toAddress,
			ToDecimals:   toDecimals,
			Amount:       amount,
			Sender:       sender,
			Destination:  destination,
		}

		bundle, err := svc.GetSwapTxBundle(ctx, params)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("swap failed: %v", err)), nil
		}

		result := swapResult{
			Chain:          fromChain,
			Provider:       bundle.Provider,
			ExpectedOutput: bundle.ExpectedOutput.String(),
			MinimumOutput:  bundle.MinimumOutput.String(),
			NeedsApproval:  bundle.NeedsApproval,
			Memo:           bundle.Memo,
			FromChain:      fromChain,
			FromSymbol:     fromSymbol,
			FromAddress:    fromAddress,
			FromDecimals:   fromDecimals,
			ToChain:        toChain,
			ToSymbol:       toSymbol,
			ToAddress:      toAddress,
			ToDecimals:     toDecimals,
			Amount:         amountStr,
			Sender:         sender,
			Destination:    destination,
		}

		swapTx := txDataToJSON(bundle.SwapTx, fromChain)
		result.SwapTx, _ = json.Marshal(swapTx)

		if bundle.NeedsApproval && bundle.ApprovalTx != nil {
			approvalTx := txDataToJSON(bundle.ApprovalTx, fromChain)
			result.ApprovalTx, _ = json.Marshal(approvalTx)
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal swap result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func txDataToJSON(tx *swap.TxData, chain string) swapTxJSON {
	result := swapTxJSON{
		To:       tx.To,
		GasLimit: tx.GasLimit,
		Memo:     tx.Memo,
	}
	if tx.Value != nil {
		result.Value = tx.Value.String()
	} else {
		result.Value = "0"
	}
	if len(tx.Data) > 0 {
		if chain == "Solana" {
			result.Data = base64.StdEncoding.EncodeToString(tx.Data)
		} else {
			result.Data = fmt.Sprintf("0x%x", tx.Data)
		}
	}
	return result
}
