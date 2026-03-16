package tornadocash

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewGetPoolsTool creates the tornado_get_pools tool definition.
func NewGetPoolsTool() mcp.Tool {
	return mcp.NewTool("tornado_get_pools",
		mcp.WithDescription(
			"List available Tornado Cash mixer pools for a given chain. "+
				"Returns pool addresses, denominations, and native token info. "+
				"Optionally filter by denomination.",
		),
		mcp.WithString("chain",
			mcp.Description("EVM chain name"),
			mcp.Required(),
			mcp.Enum(SupportedChains...),
		),
		mcp.WithString("denomination",
			mcp.Description("Filter by denomination (e.g. \"0.1\", \"1\", \"10\", \"100\"). Omit to list all."),
		),
	)
}

type poolSummary struct {
	Chain        string `json:"chain"`
	Token        string `json:"token"`
	Denomination string `json:"denomination"`
	Address      string `json:"address"`
	ValueWei     string `json:"value_wei"`
}

// HandleGetPools returns the handler for tornado_get_pools.
func HandleGetPools() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chain, err := req.RequireString("chain")
		if err != nil {
			return mcp.NewToolResultError("chain parameter is required"), nil
		}

		denomination := req.GetString("denomination", "")
		pools := ListPools(chain, denomination)
		if len(pools) == 0 {
			return mcp.NewToolResultError(fmt.Sprintf("no pools found for chain %q denomination %q", chain, denomination)), nil
		}

		summaries := make([]poolSummary, len(pools))
		for i, p := range pools {
			summaries[i] = poolSummary{
				Chain:        p.Chain,
				Token:        p.Token,
				Denomination: p.Denomination,
				Address:      p.Address.Hex(),
				ValueWei:     p.ValueWei.String(),
			}
		}

		data, err := json.Marshal(summaries)
		if err != nil {
			return nil, fmt.Errorf("marshal pools: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
