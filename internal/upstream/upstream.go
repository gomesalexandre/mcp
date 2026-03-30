// Package upstream proxies tools from external MCP servers into our server.
// It spawns each upstream as a child process over stdio, discovers tools
// via the MCP protocol, and re-registers them as native tools with an
// optional name prefix (e.g. "pendle_get_markets").
package upstream

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/config"
	"github.com/vultisig/mcp/internal/toolmeta"
)

// Upstream wraps a single external MCP server connection.
type Upstream struct {
	Name   string
	Client *client.Client
	Logger *log.Logger
}

// StartAll spawns all configured upstream MCP servers, discovers their
// tools, and registers proxy handlers on s.
func StartAll(ctx context.Context, s *server.MCPServer, configs []config.UpstreamConfig, logger *log.Logger) []*Upstream {
	var running []*Upstream
	for _, cfg := range configs {
		u, err := start(ctx, s, cfg, logger)
		if err != nil {
			logger.Printf("[WARN] upstream %s: failed to start: %v", cfg.Name, err)
			continue
		}
		running = append(running, u)
	}
	return running
}

func start(ctx context.Context, s *server.MCPServer, cfg config.UpstreamConfig, logger *log.Logger) (*Upstream, error) {
	c, err := client.NewStdioMCPClient(cfg.Command, cfg.Env, cfg.Args...)
	if err != nil {
		return nil, fmt.Errorf("spawn %s %v: %w", cfg.Command, cfg.Args, err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "vultisig-mcp-proxy",
		Version: "0.1.0",
	}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		c.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	listResp, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("list tools: %w", err)
	}

	u := &Upstream{
		Name:   cfg.Name,
		Client: c,
		Logger: logger,
	}

	for _, tool := range listResp.Tools {
		registerProxiedTool(u, s, tool, cfg.Name, cfg.Prefix)
	}

	logger.Printf("upstream %s: registered %d tools", cfg.Name, len(listResp.Tools))
	return u, nil
}

// registerProxiedTool re-registers a discovered upstream tool as a passthrough.
// If prefix is non-empty, the tool is renamed to prefix_originalName.
func registerProxiedTool(u *Upstream, s *server.MCPServer, tool mcp.Tool, category string, prefix string) {
	originalName := tool.Name

	if prefix != "" {
		tool.Name = prefix + "_" + tool.Name
	}

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fwd := mcp.CallToolRequest{}
		fwd.Params.Name = originalName
		fwd.Params.Arguments = req.Params.Arguments
		return u.Client.CallTool(ctx, fwd)
	}
	toolmeta.Register(s, tool, handler, category)
}

// Close shuts down the upstream subprocess.
func (u *Upstream) Close() error {
	u.Logger.Printf("upstream %s: shutting down", u.Name)
	return u.Client.Close()
}

// CloseAll shuts down all upstreams.
func CloseAll(upstreams []*Upstream) {
	for _, u := range upstreams {
		if err := u.Close(); err != nil {
			u.Logger.Printf("[WARN] upstream %s: close error: %v", u.Name, err)
		}
	}
}
