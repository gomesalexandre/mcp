package tornadocash

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewGenerateNoteTool creates the tornado_generate_note tool definition.
func NewGenerateNoteTool() mcp.Tool {
	return mcp.NewTool("tornado_generate_note",
		mcp.WithDescription(
			"Generate a fresh Tornado Cash note (random secret + nullifier) and compute the Pedersen commitment. "+
				"The returned note_data MUST be stored securely via the protocol-states API — losing it means permanently locked funds. "+
				"Never reveal the raw note data in chat.",
		),
	)
}

type noteResult struct {
	Commitment string   `json:"commitment"`
	NoteData   noteData `json:"note_data"`
}

type noteData struct {
	Secret        string `json:"secret"`
	Nullifier     string `json:"nullifier"`
	Commitment    string `json:"commitment"`
	NullifierHash string `json:"nullifier_hash"`
}

// HandleGenerateNote returns the handler for tornado_generate_note.
func HandleGenerateNote() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		secret, nullifier, err := GenerateNote()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("generate note: %v", err)), nil
		}

		commitment, err := PedersenCommitment(nullifier, secret)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("compute commitment: %v", err)), nil
		}

		nh, err := NullifierHash(nullifier)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("compute nullifier hash: %v", err)), nil
		}

		commitmentHex := fmt.Sprintf("0x%064x", commitment)
		nhHex := fmt.Sprintf("0x%064x", nh)

		result := noteResult{
			Commitment: commitmentHex,
			NoteData: noteData{
				Secret:        "0x" + hex.EncodeToString(secret),
				Nullifier:     "0x" + hex.EncodeToString(nullifier),
				Commitment:    commitmentHex,
				NullifierHash: nhHex,
			},
		}

		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal note: %w", err)
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}
