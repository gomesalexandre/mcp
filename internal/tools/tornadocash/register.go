package tornadocash

import (
	"github.com/mark3labs/mcp-go/server"

	evmclient "github.com/vultisig/mcp/internal/evm"
	"github.com/vultisig/mcp/internal/toolmeta"
)

// RegisterAll adds all Tornado Cash MCP tools to the server.
func RegisterAll(s *server.MCPServer, pool *evmclient.Pool) {
	toolmeta.Register(s, NewGetPoolsTool(), HandleGetPools(), "tornado")
	toolmeta.Register(s, NewGenerateNoteTool(), HandleGenerateNote(), "tornado")
	toolmeta.Register(s, NewComputeNullifierHashTool(), HandleComputeNullifierHash(), "tornado")
	toolmeta.Register(s, NewDepositTool(), HandleDeposit(), "tornado", "send")
	toolmeta.Register(s, NewWithdrawTool(), HandleWithdraw(), "tornado", "send")
	toolmeta.Register(s, NewCheckDepositTool(), HandleCheckDeposit(pool), "tornado", "contract")
	toolmeta.Register(s, NewCheckSpentTool(), HandleCheckSpent(pool), "tornado", "contract")
	toolmeta.Register(s, NewGetMerklePathTool(), HandleGetMerklePath(pool), "tornado", "contract")
}
