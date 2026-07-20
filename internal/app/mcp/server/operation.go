package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeContext(ctx context.Context, req *mcp.CallToolRequest, toolName string, input any) context.Context {
	return contract.WithWriteOperation(ctx, operationID(req, toolName, input))
}

func operationID(req *mcp.CallToolRequest, toolName string, input any) string {
	var sessionID, hostOperationID string
	if req != nil {
		if req.GetSession() != nil {
			sessionID = req.GetSession().ID()
		}
		if req.Params.Meta != nil {
			hostOperationID, _ = req.Params.Meta["idempotency_key"].(string)
		}
	}
	return operationIDFor(sessionID, hostOperationID, toolName, input)
}

func operationIDFor(sessionID, hostOperationID, toolName string, input any) string {
	hostOperationID = strings.TrimSpace(hostOperationID)
	if hostOperationID != "" && len(hostOperationID) <= 200 {
		return hostOperationID
	}
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(toolName) == "" {
		return randomOperationID()
	}
	canonicalInput, err := json.Marshal(input)
	if err != nil {
		return randomOperationID()
	}
	sum := sha256.Sum256([]byte(sessionID + "\x00" + toolName + "\x00" + string(canonicalInput)))
	return "mcp:" + hex.EncodeToString(sum[:])
}

func randomOperationID() string {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "mcp"
	}
	return "mcp:" + hex.EncodeToString(value)
}
