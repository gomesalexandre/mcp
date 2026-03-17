package tornadocash

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewComputeNullifierHashTool creates the tornado_compute_nullifier_hash tool definition.
func NewComputeNullifierHashTool() mcp.Tool {
	return mcp.NewTool("tornado_compute_nullifier_hash",
		mcp.WithDescription(
			"Compute the Pedersen nullifier hash from a raw nullifier value. "+
				"Used to check whether a note has been spent on-chain.",
		),
		mcp.WithString("nullifier",
			mcp.Description("Hex-encoded nullifier (31 bytes, with or without 0x prefix)."),
			mcp.Required(),
		),
	)
}

// HandleComputeNullifierHash returns the handler for tornado_compute_nullifier_hash.
func HandleComputeNullifierHash() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		nullifierHex, err := req.RequireString("nullifier")
		if err != nil {
			return mcp.NewToolResultError("nullifier parameter is required"), nil
		}

		nullifierBytes, err := hex.DecodeString(strings.TrimPrefix(nullifierHex, "0x"))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid nullifier hex: %v", err)), nil
		}
		if len(nullifierBytes) != 31 {
			return mcp.NewToolResultError(fmt.Sprintf("nullifier must be 31 bytes, got %d", len(nullifierBytes))), nil
		}

		nh, err := NullifierHash(nullifierBytes)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("compute nullifier hash: %v", err)), nil
		}

		result := map[string]string{
			"nullifier_hash": fmt.Sprintf("0x%064x", nh),
		}
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal nullifier hash: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
