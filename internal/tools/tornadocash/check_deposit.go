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

var commitmentsSelector = crypto.Keccak256([]byte("commitments(bytes32)"))[:4]

// NewCheckDepositTool creates the tornado_check_deposit tool definition.
func NewCheckDepositTool() mcp.Tool {
	return mcp.NewTool("tornado_check_deposit",
		mcp.WithDescription(
			"Check whether a commitment has been deposited into a Tornado Cash pool. "+
				"Queries the on-chain commitments mapping.",
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
		mcp.WithString("commitment",
			mcp.Description("Bytes32 commitment hash (0x-prefixed, 66 chars)."),
			mcp.Required(),
		),
	)
}

// HandleCheckDeposit returns the handler for tornado_check_deposit.
func HandleCheckDeposit(pool *evmclient.Pool) server.ToolHandlerFunc {
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

		commitmentHex, err := req.RequireString("commitment")
		if err != nil {
			return mcp.NewToolResultError("commitment parameter is required"), nil
		}
		commitmentBytes, err := hex.DecodeString(strings.TrimPrefix(commitmentHex, "0x"))
		if err != nil || len(commitmentBytes) != 32 {
			return mcp.NewToolResultError("commitment must be 32 bytes hex"), nil
		}

		client, _, err := pool.Get(ctx, chain)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("chain %s unavailable: %v", chain, err)), nil
		}

		calldata := make([]byte, 4+32)
		copy(calldata[:4], commitmentsSelector)
		copy(calldata[4:], commitmentBytes)

		output, err := client.CallContract(ctx, ethereum.CallMsg{
			To:   &poolAddr,
			Data: calldata,
		}, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("eth_call failed: %v", err)), nil
		}

		deposited := new(big.Int).SetBytes(output).Sign() != 0

		result := map[string]any{
			"deposited":  deposited,
			"commitment": commitmentHex,
			"pool":       poolAddr.Hex(),
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal check deposit: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
