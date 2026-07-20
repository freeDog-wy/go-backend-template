package server

import (
	"context"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type taxonomyInput struct {
	Locale  string `json:"locale" jsonschema:"locale to query"`
	Page    int    `json:"page,omitempty" jsonschema:"page number, default 1"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"items per page, maximum 100"`
}

type tagCreateInput struct {
	Locale string `json:"locale" jsonschema:"tag locale"`
	Name   string `json:"name" jsonschema:"tag name"`
	Slug   string `json:"slug" jsonschema:"tag slug"`
}

type tagTranslationInput struct {
	TagID  uint   `json:"tag_id" jsonschema:"tag ID"`
	Locale string `json:"locale" jsonschema:"translation locale"`
	Name   string `json:"name" jsonschema:"tag name"`
	Slug   string `json:"slug" jsonschema:"tag slug"`
}

func registerTagTools(server *mcp.Server, client contract.TagService, annotations toolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{Name: "cms.tag.list", Description: "List CMS tags for one locale.", Annotations: annotations.readOnly}, func(ctx context.Context, _ *mcp.CallToolRequest, input taxonomyInput) (*mcp.CallToolResult, map[string]any, error) {
		if strings.TrimSpace(input.Locale) == "" {
			return toolError("INVALID_INPUT", "locale is required"), nil, nil
		}
		return toolOutput(client.Tags(ctx, input.Locale, input.Page, input.PerPage))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.tag.create", Description: "Create one tag and its initial translation. Confirm the tag fields with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input tagCreateInput) (*mcp.CallToolResult, map[string]any, error) {
		if err := validateNamedTranslation(input.Locale, input.Name, input.Slug); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		return toolOutput(client.CreateTag(writeContext(ctx, req, "cms.tag.create", input), contract.TagInput{Locale: input.Locale, Name: input.Name, Slug: input.Slug}))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.tag.upsert_translation", Description: "Create or update one tag translation. Confirm the translation fields with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input tagTranslationInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.TagID == 0 {
			return toolError("INVALID_INPUT", "tag_id is required"), nil, nil
		}
		if err := validateNamedTranslation(input.Locale, input.Name, input.Slug); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		return toolOutput(client.UpsertTagTranslation(writeContext(ctx, req, "cms.tag.upsert_translation", input), input.TagID, input.Locale, contract.TagTranslationInput{Name: input.Name, Slug: input.Slug}))
	})
}
