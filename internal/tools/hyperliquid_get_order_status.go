package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/hyperliquid"
	"github.com/vultisig/mcp/internal/resolve"
	"github.com/vultisig/mcp/internal/vault"
)

func newHyperliquidGetOrderStatusTool() mcp.Tool {
	return mcp.NewTool("hyperliquid_get_order_status",
		mcp.WithDescription(
			"Query the status of a specific order on Hyperliquid by order ID or client order ID. "+
				"Returns order details and status (filled, resting, cancelled, triggered, rejected, marginCanceled). "+
				"Provide exactly one of oid or cloid. "+
				"Accepts inline vault keys (ecdsa_public_key, eddsa_public_key, chain_code) or falls back to set_vault_info session state.",
		),
		mcp.WithString("address",
			mcp.Description("Ethereum-format address (0x-prefixed). Optional if vault info is set."),
		),
		mcp.WithNumber("oid",
			mcp.Description("Order ID (from hyperliquid_get_open_orders). Provide oid or cloid, not both."),
		),
		mcp.WithString("cloid",
			mcp.Description("Client order ID. Provide oid or cloid, not both."),
		),
	)
}

func handleHyperliquidGetOrderStatus(store *vault.Store, hlClient *hyperliquid.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		explicit := req.GetString("address", "")

		addr, err := resolve.EVMAddress(explicit, resolve.ResolveVault(ctx, req, store))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		oid := int64(req.GetFloat("oid", 0))
		cloid := req.GetString("cloid", "")

		if oid == 0 && cloid == "" {
			return mcp.NewToolResultError("provide exactly one of oid or cloid"), nil
		}
		if oid != 0 && cloid != "" {
			return mcp.NewToolResultError("provide exactly one of oid or cloid, not both"), nil
		}

		var status *hyperliquid.OrderStatusResponse
		if oid != 0 {
			status, err = hlClient.GetOrderStatusByOid(ctx, addr, oid)
		} else {
			status, err = hlClient.GetOrderStatusByCloid(ctx, addr, cloid)
		}
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get order status: %v", err)), nil
		}

		result := map[string]any{
			"address": addr,
			"status":  status.Status,
			"order":   status.Order,
		}

		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
