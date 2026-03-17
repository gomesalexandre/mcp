package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/recipes/sdk/bridge"

	evmclient "github.com/vultisig/mcp/internal/evm"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

var bridgeApproveABI abi.ABI

func init() {
	var err error
	bridgeApproveABI, err = abi.JSON(strings.NewReader(`[{"type":"function","name":"approve","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`))
	if err != nil {
		panic(fmt.Sprintf("parse approve ABI: %v", err))
	}
}

func newBridgeAssetTool() mcp.Tool {
	return mcp.NewTool("bridge_asset",
		mcp.WithDescription(
			"Build unsigned transaction(s) to bridge an asset between chains. "+
				"Supports native tokens (ETH) and ERC20 tokens (USDC, etc.) via multiple providers: "+
				"native L2 bridges (Arbitrum, Optimism, Base), LiFi, Across, and deBridge. "+
				"Returns the bridge transaction and an optional ERC20 approval transaction. "+
				"For ERC20 tokens, both approval and bridge transactions must be signed and submitted sequentially.",
		),
		mcp.WithString("from_chain",
			mcp.Description("Source chain (e.g. 'Ethereum', 'Arbitrum', 'Optimism', 'Base')"),
			mcp.Required(),
		),
		mcp.WithString("to_chain",
			mcp.Description("Destination chain (e.g. 'Arbitrum', 'Ethereum', 'Optimism', 'Base')"),
			mcp.Required(),
		),
		mcp.WithString("symbol",
			mcp.Description("Token symbol (e.g. 'ETH', 'USDC')"),
			mcp.Required(),
		),
		mcp.WithString("token_address",
			mcp.Description("Token contract address on source chain (0x-prefixed). Leave empty for native tokens (ETH)."),
		),
		mcp.WithNumber("decimals",
			mcp.Description("Token decimals (e.g. 18 for ETH, 6 for USDC)"),
			mcp.Required(),
		),
		mcp.WithString("amount",
			mcp.Description("Amount in human-readable units (e.g. '10' for 10 USDC or '0.1' for 0.1 ETH)"),
			mcp.Required(),
		),
		mcp.WithString("address",
			mcp.Description("Sender/recipient address (0x-prefixed). Same address used on both chains. Optional if vault info is set."),
		),
		mcp.WithString("destination_token_address",
			mcp.Description("Token contract address on destination chain. Defaults to same as source if omitted."),
		),
	)
}

func handleBridgeAsset(store *vault.Store, pool *evmclient.Pool) server.ToolHandlerFunc {
	router := bridge.NewDefaultRouter()
	erc20Router := bridge.NewRouter(
		bridge.WithProvider(bridge.NewLiFiProvider("")),
		bridge.WithProvider(bridge.NewAcrossProvider()),
		bridge.WithProvider(bridge.NewDeBridgeProvider()),
	)

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fromChain, err := req.RequireString("from_chain")
		if err != nil {
			return mcp.NewToolResultError("from_chain is required"), nil
		}
		toChain, err := req.RequireString("to_chain")
		if err != nil {
			return mcp.NewToolResultError("to_chain is required"), nil
		}
		if fromChain == toChain {
			return mcp.NewToolResultError("from_chain and to_chain must be different"), nil
		}

		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError("symbol is required"), nil
		}

		tokenAddress := req.GetString("token_address", "")
		decimals := int(req.GetInt("decimals", 18))

		amountStr, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("amount is required"), nil
		}

		amountFloat, ok := new(big.Float).SetString(amountStr)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("invalid amount: %s", amountStr)), nil
		}
		if amountFloat.Sign() <= 0 {
			return mcp.NewToolResultError("amount must be positive"), nil
		}

		multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
		rawAmount, _ := new(big.Float).Mul(amountFloat, multiplier).Int(nil)

		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		destTokenAddress := req.GetString("destination_token_address", tokenAddress)

		fromAsset := bridge.BridgeAsset{
			Chain:    fromChain,
			Symbol:   symbol,
			Address:  tokenAddress,
			Decimals: decimals,
		}
		toAsset := bridge.BridgeAsset{
			Chain:    toChain,
			Symbol:   symbol,
			Address:  destTokenAddress,
			Decimals: decimals,
		}

		activeRouter := router
		if tokenAddress != "" {
			activeRouter = erc20Router
		}

		quote, err := activeRouter.GetQuote(ctx, bridge.QuoteRequest{
			From:        fromAsset,
			To:          toAsset,
			Amount:      rawAmount,
			Sender:      addr,
			Destination: addr,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("no bridge route available: %v", err)), nil
		}

		bridgeResult, err := activeRouter.BuildTx(ctx, bridge.BridgeRequest{
			Quote:       quote,
			Sender:      addr,
			Destination: addr,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("build bridge tx failed: %v", err)), nil
		}

		senderAddr := common.HexToAddress(addr)
		client, chainID, err := pool.Get(ctx, fromChain)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("%s chain unavailable: %v", fromChain, err)), nil
		}

		nonce, err := client.PendingNonce(ctx, senderAddr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get nonce: %v", err)), nil
		}

		tipCap, err := client.SuggestGasTipCap(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get gas tip cap: %v", err)), nil
		}

		baseFee, err := client.LatestBaseFee(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get base fee: %v", err)), nil
		}

		maxFee := new(big.Int).Mul(baseFee, big.NewInt(2))
		maxFee.Add(maxFee, tipCap)

		result := map[string]any{
			"provider":        bridgeResult.Provider,
			"from_chain":      fromChain,
			"to_chain":        toChain,
			"symbol":          symbol,
			"amount":          amountStr,
			"amount_raw":      rawAmount.String(),
			"expected_output": bridgeResult.ExpectedOut.String(),
			"address":         addr,
			"needs_approval":  bridgeResult.NeedsApproval,
		}

		currentNonce := nonce

		if bridgeResult.NeedsApproval {
			approveData, abiErr := bridgeApproveABI.Pack(
				"approve",
				common.HexToAddress(bridgeResult.ApprovalAddress),
				bridgeResult.ApprovalAmount,
			)
			if abiErr != nil {
				return mcp.NewToolResultError(fmt.Sprintf("encode approve: %v", abiErr)), nil
			}

			tokenAddr := common.HexToAddress(tokenAddress)
			approveGas := uint64(60000)
			estimatedGas, gasErr := client.EstimateGas(ctx, ethereum.CallMsg{
				From: senderAddr,
				To:   &tokenAddr,
				Data: approveData,
			})
			if gasErr == nil {
				approveGas = estimatedGas + estimatedGas/5
			}

			result["approval_tx"] = map[string]any{
				"chain":                    fromChain,
				"chain_id":                 chainID.String(),
				"to":                       tokenAddr.Hex(),
				"value":                    "0",
				"data":                     fmt.Sprintf("0x%x", approveData),
				"nonce":                    fmt.Sprintf("%d", currentNonce),
				"gas_limit":                fmt.Sprintf("%d", approveGas),
				"max_fee_per_gas":          maxFee.String(),
				"max_priority_fee_per_gas": tipCap.String(),
				"tx_type":                  2,
			}
			currentNonce++
		}

		bridgeValue := "0"
		if bridgeResult.Value != nil {
			bridgeValue = bridgeResult.Value.String()
		}

		bridgeGas := uint64(250000)
		toAddr := common.HexToAddress(bridgeResult.ToAddress)
		callMsg := ethereum.CallMsg{
			From: senderAddr,
			To:   &toAddr,
			Data: bridgeResult.TxData,
		}
		if bridgeResult.Value != nil && bridgeResult.Value.Sign() > 0 {
			callMsg.Value = bridgeResult.Value
		}
		estimatedGas, gasErr := client.EstimateGas(ctx, callMsg)
		if gasErr == nil {
			bridgeGas = estimatedGas + estimatedGas/5
		}

		result["bridge_tx"] = map[string]any{
			"chain":                    fromChain,
			"chain_id":                 chainID.String(),
			"to":                       toAddr.Hex(),
			"value":                    bridgeValue,
			"data":                     fmt.Sprintf("0x%x", bridgeResult.TxData),
			"nonce":                    fmt.Sprintf("%d", currentNonce),
			"gas_limit":                fmt.Sprintf("%d", bridgeGas),
			"max_fee_per_gas":          maxFee.String(),
			"max_priority_fee_per_gas": tipCap.String(),
			"tx_type":                  2,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
