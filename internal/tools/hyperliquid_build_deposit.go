package tools

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	evmclient "github.com/vultisig/mcp/internal/evm"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

const (
	arbitrumChainID                  = 42161
	hlBridgeAddress                  = "0x2Df1c51E09aECF9cacB7bc98cB1742757f163dF7"
	arbitrumUSDCAddress              = "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"
	usdcDecimals                     = 6
	minDepositUSD                    = 5
	permitDeadlineHours              = 1
	noncesSelector                   = "7ecebe00"
	batchedDepositWithPermitSelector = "02c7c038"
)

func newHyperliquidBuildDepositTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_build_deposit",
		mcp.WithDescription(
			"Build an unsigned Hyperliquid deposit via the Arbitrum bridge. "+
				"Returns the EIP-2612 permit typed data to sign and the Arbitrum bridge transaction calldata. "+
				"The caller must: (1) sign the permit EIP-712 typed data, (2) encode the bridge calldata "+
				"with the permit signature, (3) submit the transaction to Arbitrum. "+
				"Minimum deposit is 5 USDC. The bridge contract is on Arbitrum One (chain ID 42161).",
		),
		mcp.WithString("amount",
			mcp.Description("Amount in USDC to deposit (e.g. '10' for 10 USDC). Minimum 5 USDC."),
			mcp.Required(),
		),
		mcp.WithString("address",
			mcp.Description("Depositor's Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
	)
}

func handleHyperliquidBuildDeposit(store *vault.Store, pool *evmclient.Pool) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")
		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		amountStr, err := req.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultError("amount is required"), nil
		}

		amountFloat, ok := new(big.Float).SetString(amountStr)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("invalid amount: %s", amountStr)), nil
		}

		minDeposit := new(big.Float).SetInt64(minDepositUSD)
		if amountFloat.Cmp(minDeposit) < 0 {
			return mcp.NewToolResultError(fmt.Sprintf("minimum deposit is %d USDC", minDepositUSD)), nil
		}

		multiplier := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(usdcDecimals), nil))
		rawAmount, _ := new(big.Float).Mul(amountFloat, multiplier).Int(nil)

		if rawAmount.BitLen() > 64 {
			return mcp.NewToolResultError("amount too large for uint64"), nil
		}
		rawUSD := rawAmount.Uint64()

		client, _, err := pool.Get(ctx, "Arbitrum")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Arbitrum chain unavailable: %v", err)), nil
		}

		ownerAddr := common.HexToAddress(addr)
		usdcAddr := common.HexToAddress(arbitrumUSDCAddress)

		permitNonce, err := readPermitNonce(ctx, client, usdcAddr, ownerAddr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read permit nonce: %v", err)), nil
		}

		deadline := time.Now().Add(permitDeadlineHours * time.Hour).Unix()

		permitTypedData := map[string]any{
			"types": map[string]any{
				"EIP712Domain": []map[string]string{
					{"name": "name", "type": "string"},
					{"name": "version", "type": "string"},
					{"name": "chainId", "type": "uint256"},
					{"name": "verifyingContract", "type": "address"},
				},
				"Permit": []map[string]string{
					{"name": "owner", "type": "address"},
					{"name": "spender", "type": "address"},
					{"name": "value", "type": "uint256"},
					{"name": "nonce", "type": "uint256"},
					{"name": "deadline", "type": "uint256"},
				},
			},
			"primaryType": "Permit",
			"domain": map[string]any{
				"name":              "USD Coin",
				"version":           "2",
				"chainId":           arbitrumChainID,
				"verifyingContract": arbitrumUSDCAddress,
			},
			"message": map[string]any{
				"owner":    addr,
				"spender":  hlBridgeAddress,
				"value":    rawAmount.String(),
				"nonce":    fmt.Sprintf("%d", permitNonce),
				"deadline": fmt.Sprintf("%d", deadline),
			},
		}

		result := map[string]any{
			"chain":                    "Hyperliquid",
			"action":                   "deposit",
			"bridge_chain":             "Arbitrum",
			"bridge_chain_id":          arbitrumChainID,
			"bridge_contract":          hlBridgeAddress,
			"usdc_contract":            arbitrumUSDCAddress,
			"address":                  addr,
			"amount":                   amountStr,
			"amount_raw":               rawAmount.String(),
			"amount_raw_uint64":        rawUSD,
			"deadline":                 deadline,
			"permit_nonce":             permitNonce,
			"permit_typed_data":        permitTypedData,
			"bridge_function":          "batchedDepositWithPermit",
			"bridge_function_selector": batchedDepositWithPermitSelector,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func readPermitNonce(ctx context.Context, client *evmclient.Client, token, owner common.Address) (uint64, error) {
	selector, err := hex.DecodeString(noncesSelector)
	if err != nil {
		return 0, fmt.Errorf("decode selector: %w", err)
	}
	data := append(selector, common.LeftPadBytes(owner.Bytes(), 32)...)

	result, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &token,
		Data: data,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("call nonces(): %w", err)
	}

	if len(result) < 32 {
		return 0, fmt.Errorf("unexpected nonces() response length: %d", len(result))
	}

	nonce := new(big.Int).SetBytes(result[:32])
	return nonce.Uint64(), nil
}
