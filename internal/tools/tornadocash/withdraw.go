package tornadocash

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// withdraw(bytes _proof, bytes32 _root, bytes32 _nullifierHash, address _recipient, address _relayer, uint256 _fee, uint256 _refund)
var withdrawSelector = crypto.Keccak256([]byte("withdraw(bytes,bytes32,bytes32,address,address,uint256,uint256)"))[:4]

// NewWithdrawTool creates the tornado_withdraw tool definition.
func NewWithdrawTool() mcp.Tool {
	return mcp.NewTool("tornado_withdraw",
		mcp.WithDescription(
			"Prepare parameters for a Tornado Cash withdrawal. Returns pool address and ABI-encoded calldata "+
				"for withdraw(bytes,bytes32,bytes32,address,address,uint256,uint256). "+
				"The proof must be generated externally (e.g. via snarkjs). "+
				"The caller must use evm_tx_info and build_evm_tx to construct the transaction with value=0.",
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
		mcp.WithString("proof",
			mcp.Description("Hex-encoded zk-SNARK proof bytes (0x-prefixed)."),
			mcp.Required(),
		),
		mcp.WithString("root",
			mcp.Description("Bytes32 Merkle root (0x-prefixed, 66 chars)."),
			mcp.Required(),
		),
		mcp.WithString("nullifier_hash",
			mcp.Description("Bytes32 nullifier hash (0x-prefixed, 66 chars)."),
			mcp.Required(),
		),
		mcp.WithString("recipient",
			mcp.Description("Recipient address for the withdrawn funds (0x-prefixed)."),
			mcp.Required(),
		),
		mcp.WithString("relayer",
			mcp.Description("Relayer address (0x-prefixed). Use 0x0000000000000000000000000000000000000000 for no relayer."),
			mcp.Required(),
		),
		mcp.WithString("fee",
			mcp.Description("Relayer fee in wei (decimal string). Use \"0\" if no relayer."),
			mcp.Required(),
		),
		mcp.WithString("refund",
			mcp.Description("Refund amount in wei (decimal string). Use \"0\" for native token withdrawals."),
			mcp.Required(),
		),
	)
}

// HandleWithdraw returns the handler for tornado_withdraw.
func HandleWithdraw() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chain, err := req.RequireString("chain")
		if err != nil {
			return mcp.NewToolResultError("chain parameter is required"), nil
		}

		poolAddrStr, err := req.RequireString("pool_address")
		if err != nil {
			return mcp.NewToolResultError("pool_address parameter is required"), nil
		}
		if !ethcommon.IsHexAddress(poolAddrStr) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid pool_address: %s", poolAddrStr)), nil
		}
		poolAddr := ethcommon.HexToAddress(poolAddrStr)

		proofHex, err := req.RequireString("proof")
		if err != nil {
			return mcp.NewToolResultError("proof parameter is required"), nil
		}
		proofBytes, err := hex.DecodeString(strings.TrimPrefix(proofHex, "0x"))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid proof hex: %v", err)), nil
		}

		rootHex, err := req.RequireString("root")
		if err != nil {
			return mcp.NewToolResultError("root parameter is required"), nil
		}
		rootBytes, err := decodeBytes32(rootHex)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid root: %v", err)), nil
		}

		nhHex, err := req.RequireString("nullifier_hash")
		if err != nil {
			return mcp.NewToolResultError("nullifier_hash parameter is required"), nil
		}
		nhBytes, err := decodeBytes32(nhHex)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid nullifier_hash: %v", err)), nil
		}

		recipientStr, err := req.RequireString("recipient")
		if err != nil {
			return mcp.NewToolResultError("recipient parameter is required"), nil
		}
		if !ethcommon.IsHexAddress(recipientStr) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid recipient: %s", recipientStr)), nil
		}

		relayerStr, err := req.RequireString("relayer")
		if err != nil {
			return mcp.NewToolResultError("relayer parameter is required"), nil
		}
		if !ethcommon.IsHexAddress(relayerStr) {
			return mcp.NewToolResultError(fmt.Sprintf("invalid relayer: %s", relayerStr)), nil
		}

		feeStr, err := req.RequireString("fee")
		if err != nil {
			return mcp.NewToolResultError("fee parameter is required"), nil
		}
		fee, ok := new(big.Int).SetString(feeStr, 10)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("invalid fee: %s", feeStr)), nil
		}
		if fee.Sign() < 0 {
			return mcp.NewToolResultError("fee must not be negative"), nil
		}

		refundStr, err := req.RequireString("refund")
		if err != nil {
			return mcp.NewToolResultError("refund parameter is required"), nil
		}
		refund, ok := new(big.Int).SetString(refundStr, 10)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("invalid refund: %s", refundStr)), nil
		}
		if refund.Sign() < 0 {
			return mcp.NewToolResultError("refund must not be negative"), nil
		}

		// ABI encode: withdraw(bytes _proof, bytes32 _root, bytes32 _nullifierHash,
		//                       address _recipient, address _relayer, uint256 _fee, uint256 _refund)
		calldata := encodeWithdrawCalldata(proofBytes, rootBytes, nhBytes,
			ethcommon.HexToAddress(recipientStr), ethcommon.HexToAddress(relayerStr), fee, refund)

		result := map[string]string{
			"pool_address":       poolAddr.Hex(),
			"function_signature": "withdraw(bytes,bytes32,bytes32,address,address,uint256,uint256)",
			"calldata":           "0x" + hex.EncodeToString(calldata),
			"value_wei":          "0",
			"chain":              chain,
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal withdraw params: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func decodeBytes32(hexStr string) ([]byte, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(hexStr, "0x"))
	if err != nil {
		return nil, err
	}
	if len(b) != 32 {
		return nil, fmt.Errorf("expected 32 bytes, got %d", len(b))
	}
	return b, nil
}

// encodeWithdrawCalldata ABI-encodes the withdraw function call.
// bytes is a dynamic type, so the encoding uses an offset pointer.
func encodeWithdrawCalldata(proof, root, nullifierHash []byte, recipient, relayer ethcommon.Address, fee, refund *big.Int) []byte {
	// 7 parameters: bytes(dynamic), bytes32, bytes32, address, address, uint256, uint256
	// Head: 7 * 32 bytes = 224 bytes for the head section
	// The first parameter (bytes) is dynamic, so its head slot contains the offset
	// to the data section. Offset = 7 * 32 = 224 = 0xE0.

	headSize := 7 * 32
	proofPadded := padRight(proof, 32)
	dataSection := make([]byte, 32+len(proofPadded)) // length + padded data
	copy(dataSection[:32], padLeft(big.NewInt(int64(len(proof))).Bytes(), 32))
	copy(dataSection[32:], proofPadded)

	result := make([]byte, 4+headSize+len(dataSection))
	copy(result[:4], withdrawSelector)

	offset := 4
	// slot 0: offset to bytes data = 0xE0 (224)
	copy(result[offset:offset+32], padLeft(big.NewInt(int64(headSize)).Bytes(), 32))
	offset += 32

	// slot 1: root (bytes32)
	copy(result[offset:offset+32], root)
	offset += 32

	// slot 2: nullifierHash (bytes32)
	copy(result[offset:offset+32], nullifierHash)
	offset += 32

	// slot 3: recipient (address, left-padded)
	copy(result[offset+12:offset+32], recipient.Bytes())
	offset += 32

	// slot 4: relayer (address, left-padded)
	copy(result[offset+12:offset+32], relayer.Bytes())
	offset += 32

	// slot 5: fee (uint256)
	copy(result[offset:offset+32], padLeft(fee.Bytes(), 32))
	offset += 32

	// slot 6: refund (uint256)
	copy(result[offset:offset+32], padLeft(refund.Bytes(), 32))
	offset += 32

	// data section: proof bytes
	copy(result[offset:], dataSection)

	return result
}

func padLeft(b []byte, size int) []byte {
	if len(b) == size {
		return b
	}
	if len(b) > size {
		// trim leading zero bytes only; values used here are uint256-safe
		return b[len(b)-size:]
	}
	result := make([]byte, size)
	copy(result[size-len(b):], b)
	return result
}

func padRight(b []byte, multiple int) []byte {
	if len(b)%multiple == 0 {
		return b
	}
	padded := make([]byte, len(b)+(multiple-len(b)%multiple))
	copy(padded, b)
	return padded
}
