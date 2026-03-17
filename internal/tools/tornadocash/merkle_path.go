package tornadocash

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	evmclient "github.com/vultisig/mcp/internal/evm"
)

// NewGetMerklePathTool creates the tornado_get_merkle_path tool definition.
func NewGetMerklePathTool() mcp.Tool {
	return mcp.NewTool("tornado_get_merkle_path",
		mcp.WithDescription(
			"Compute the Merkle proof path for a deposited commitment in a Tornado Cash pool. "+
				"Fetches all Deposit events on-chain and builds the MiMCSponge Merkle tree. "+
				"WARNING: This is a heavy operation — may take minutes on pools with many deposits. "+
				"Returns root, path_elements, and path_indices needed for proof generation.",
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
			mcp.Description("Bytes32 commitment to find in the tree (0x-prefixed, 66 chars)."),
			mcp.Required(),
		),
		mcp.WithString("from_block",
			mcp.Description("Block number to start scanning from (decimal string). Use to skip blocks before pool deployment. Default: \"0\"."),
		),
	)
}

type merklePathResult struct {
	Root         string   `json:"root"`
	PathElements []string `json:"path_elements"`
	PathIndices  []int    `json:"path_indices"`
	LeafIndex    int      `json:"leaf_index"`
	TotalLeaves  int      `json:"total_leaves"`
}

// HandleGetMerklePath returns the handler for tornado_get_merkle_path.
func HandleGetMerklePath(pool *evmclient.Pool) server.ToolHandlerFunc {
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
		commitment := new(big.Int).SetBytes(commitmentBytes)

		var fromBlock uint64
		if s := req.GetString("from_block", "0"); s != "" && s != "0" {
			fb, ok := new(big.Int).SetString(s, 10)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("invalid from_block: %s", s)), nil
			}
			fromBlock = fb.Uint64()
		}

		client, _, err := pool.Get(ctx, chain)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("chain %s unavailable: %v", chain, err)), nil
		}

		tree, err := BuildTreeFromDeposits(ctx, client.ETH(), poolAddr, fromBlock)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("build tree: %v", err)), nil
		}

		leafIndex := tree.FindLeafIndex(commitment)
		if leafIndex < 0 {
			return mcp.NewToolResultError("commitment not found in pool deposits"), nil
		}

		root, pathElements, pathIndices := tree.GetPath(leafIndex)

		elemHexes := make([]string, len(pathElements))
		for i, e := range pathElements {
			elemHexes[i] = fmt.Sprintf("0x%064x", e)
		}

		result := merklePathResult{
			Root:         fmt.Sprintf("0x%064x", root),
			PathElements: elemHexes,
			PathIndices:  pathIndices,
			LeafIndex:    leafIndex,
			TotalLeaves:  len(tree.Leaves),
		}

		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal merkle path: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
