package server

import (
	"context"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mediaListInput struct {
	Page    int `json:"page,omitempty" jsonschema:"page number, default 1"`
	PerPage int `json:"per_page,omitempty" jsonschema:"items per page, maximum 100"`
}

type mediaUploadRequestInput struct {
	Filename    string `json:"filename" jsonschema:"local filename"`
	ContentType string `json:"content_type" jsonschema:"image MIME type"`
	SizeBytes   int64  `json:"size_bytes" jsonschema:"file size in bytes"`
}

type mediaIDInput struct {
	MediaID uint `json:"media_id" jsonschema:"media ID"`
}

type mediaTranslationInput struct {
	MediaID uint   `json:"media_id" jsonschema:"media ID"`
	Locale  string `json:"locale" jsonschema:"translation locale"`
	AltText string `json:"alt_text,omitempty"`
	Title   string `json:"title,omitempty"`
}

func registerMediaTools(server *mcp.Server, client contract.MediaService, annotations toolAnnotations) {
	if client == nil {
		return
	}
	mcp.AddTool(server, &mcp.Tool{Name: "cms.media.list", Description: "List CMS media assets.", Annotations: annotations.readOnly}, func(ctx context.Context, _ *mcp.CallToolRequest, input mediaListInput) (*mcp.CallToolResult, map[string]any, error) {
		return toolOutput(client.Media(ctx, input.Page, input.PerPage))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.media.request_upload", Description: "Request a pre-signed upload URL for an image. Upload the file to the returned URL before completing it.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input mediaUploadRequestInput) (*mcp.CallToolResult, map[string]any, error) {
		if strings.TrimSpace(input.Filename) == "" || strings.TrimSpace(input.ContentType) == "" || input.SizeBytes <= 0 {
			return toolError("INVALID_INPUT", "filename, content_type, and a positive size_bytes are required"), nil, nil
		}
		return toolOutput(client.RequestMediaUpload(writeContext(ctx, req, "cms.media.request_upload", input), contract.MediaUploadRequestInput{Filename: input.Filename, ContentType: input.ContentType, SizeBytes: input.SizeBytes}))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.media.complete_upload", Description: "Validate and mark a previously uploaded media asset ready. Call only after its pre-signed upload succeeds.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input mediaIDInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.MediaID == 0 {
			return toolError("INVALID_INPUT", "media_id is required"), nil, nil
		}
		return toolOutput(client.CompleteMediaUpload(writeContext(ctx, req, "cms.media.complete_upload", input), input.MediaID))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.media.upsert_translation", Description: "Set localized alternative text and title for a media asset. Confirm the fields with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input mediaTranslationInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.MediaID == 0 || strings.TrimSpace(input.Locale) == "" {
			return toolError("INVALID_INPUT", "media_id and locale are required"), nil, nil
		}
		return toolOutput(client.UpsertMediaTranslation(writeContext(ctx, req, "cms.media.upsert_translation", input), input.MediaID, input.Locale, contract.MediaTranslationInput{AltText: input.AltText, Title: input.Title}))
	})
}
