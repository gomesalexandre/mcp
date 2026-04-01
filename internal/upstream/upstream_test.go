package upstream

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/vultisig/mcp/internal/config"
)

func TestStartAll_InvalidCommand(t *testing.T) {
	s := server.NewMCPServer("test", "0.1.0", server.WithToolCapabilities(true))
	logger := log.New(os.Stderr, "[test] ", 0)

	configs := []config.UpstreamConfig{
		{Name: "nonexistent", Command: "/nonexistent/binary", Args: []string{}},
	}

	upstreams := StartAll(context.Background(), s, configs, logger)
	if len(upstreams) != 0 {
		t.Fatalf("expected 0 upstreams for invalid command, got %d", len(upstreams))
	}
}

func TestStartAll_NoConfigs(t *testing.T) {
	s := server.NewMCPServer("test", "0.1.0", server.WithToolCapabilities(true))
	logger := log.New(os.Stderr, "[test] ", 0)

	upstreams := StartAll(context.Background(), s, nil, logger)
	if len(upstreams) != 0 {
		t.Fatalf("expected 0 upstreams for nil configs, got %d", len(upstreams))
	}
}

func TestRegisterProxiedTool_WithPrefix(t *testing.T) {
	s := server.NewMCPServer("test", "0.1.0", server.WithToolCapabilities(true))
	logger := log.New(os.Stderr, "[test] ", 0)

	u := &Upstream{Name: "test", client: nil, logger: logger}

	tool := mcp.NewTool("get_markets",
		mcp.WithDescription("list markets"),
		mcp.WithString("input", mcp.Description("input value")),
	)

	registerProxiedTool(u, s, tool, "test-category", "pendle")

	toolMap := s.ListTools()
	if _, ok := toolMap["pendle_get_markets"]; !ok {
		t.Fatal("pendle_get_markets not found in registered tools")
	}
	if _, ok := toolMap["get_markets"]; ok {
		t.Fatal("original name get_markets should not be registered")
	}
}

func TestRegisterProxiedTool_NoPrefix(t *testing.T) {
	s := server.NewMCPServer("test", "0.1.0", server.WithToolCapabilities(true))
	logger := log.New(os.Stderr, "[test] ", 0)

	u := &Upstream{Name: "test", client: nil, logger: logger}

	tool := mcp.NewTool("get_balance",
		mcp.WithDescription("get balance"),
	)

	registerProxiedTool(u, s, tool, "test-category", "")

	toolMap := s.ListTools()
	if _, ok := toolMap["get_balance"]; !ok {
		t.Fatal("get_balance not found in registered tools")
	}
}
