package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
)

func newHyperliquidGetAllMidsTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_all_mids",
		mcp.WithDescription(
			"Query mid prices for all coins on Hyperliquid. "+
				"Returns a map of coin names to their current mid prices. No address required.",
		),
	)
}

func handleHyperliquidGetAllMids(hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		mids, err := hlClient.GetAllMids(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get all mids: %v", err)), nil
		}

		data, err := json.Marshal(mids)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
