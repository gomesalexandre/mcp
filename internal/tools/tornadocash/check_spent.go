package tornadocash

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	evmclient "github.com/vultisig/mcp/internal/evm"
)

var nullifierHashesSelector = crypto.Keccak256([]byte("nullifierHashes(bytes32)"))[:4]

// NewCheckSpentTool creates the tornado_check_spent tool definition.
func NewCheckSpentTool() mcp.Tool {
	return mcp.NewTool("tornado_check_spent",
		mcp.WithDescription(
			"Check whether a nullifier hash has been spent (used in a withdrawal) on a Tornado Cash pool. "+
				"Queries the on-chain nullifierHashes mapping.",
		),
		mcp.WithString("chain",
			mcp.Description("EVM chain name"),
			mcp.Required(),
			mcp.Enum(SupportedChains...),
		),
		mcp.WithString("pool_address",
			mcp.Description("Tornado Cash pool contract address (0x-prefixed)."),
			mcp.Required(),
		),
		mcp.WithString("nullifier_hash",
			mcp.Description("Bytes32 nullifier hash (0x-prefixed, 66 chars)."),
			mcp.Required(),
		),
	)
}

// HandleCheckSpent returns the handler for tornado_check_spent.
func HandleCheckSpent(pool *evmclient.Pool) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chain, err := req.RequireString("chain")
		if err != nil {
			return mcp.NewToolResultError("chain parameter is required"), nil
		}

		poolAddrStr, err := req.RequireString("pool_address")
		if err != nil {
			return mcp.NewToolResultError("pool_address parameter is required"), nil
		}
		if !common.IsHexAddress(poolAddrStr) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid pool_address: %s", poolAddrStr)), nil
		}
		poolAddr := common.HexToAddress(poolAddrStr)

		nhHex, err := req.RequireString("nullifier_hash")
		if err != nil {
			return mcp.NewToolResultError("nullifier_hash parameter is required"), nil
		}
		nhBytes, err := hex.DecodeString(strings.TrimPrefix(nhHex, "0x"))
		if err != nil || len(nhBytes) != 32 {
			return mcp.NewToolResultError("nullifier_hash must be 32 bytes hex"), nil
		}

		client, _, err := pool.Get(ctx, chain)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("chain %s unavailable: %v", chain, err)), nil
		}

		calldata := make([]byte, 4+32)
		copy(calldata[:4], nullifierHashesSelector)
		copy(calldata[4:], nhBytes)

		output, err := client.CallContract(ctx, ethereum.CallMsg{
			To:   &poolAddr,
			Data: calldata,
		}, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("eth_call failed: %v", err)), nil
		}

		spent := new(big.Int).SetBytes(output).Sign() != 0

		result := map[string]any{
			"spent":          spent,
			"nullifier_hash": nhHex,
			"pool":           poolAddr.Hex(),
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal check spent: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
