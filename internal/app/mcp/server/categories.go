package server

import (
	"context"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type categoryCreateInput struct {
	ParentID       *uint  `json:"parent_id,omitempty" jsonschema:"optional parent category ID"`
	SortOrder      int    `json:"sort_order,omitempty"`
	Locale         string `json:"locale" jsonschema:"category locale"`
	Name           string `json:"name" jsonschema:"category name"`
	Slug           string `json:"slug" jsonschema:"category slug"`
	Description    string `json:"description,omitempty"`
	SEOTitle       string `json:"seo_title,omitempty"`
	SEODescription string `json:"seo_description,omitempty"`
}

type categoryUpdateInput struct {
	CategoryID uint `json:"category_id" jsonschema:"category ID"`
	IsEnabled  bool `json:"is_enabled"`
	SortOrder  int  `json:"sort_order,omitempty"`
}

type categoryMoveInput struct {
	CategoryID uint  `json:"category_id" jsonschema:"category ID"`
	ParentID   *uint `json:"parent_id,omitempty" jsonschema:"optional parent category ID; omit or null for root"`
	SortOrder  int   `json:"sort_order,omitempty"`
}

type categoryTranslationInput struct {
	CategoryID     uint   `json:"category_id" jsonschema:"category ID"`
	Locale         string `json:"locale" jsonschema:"translation locale"`
	Name           string `json:"name" jsonschema:"category name"`
	Slug           string `json:"slug" jsonschema:"category slug"`
	Description    string `json:"description,omitempty"`
	SEOTitle       string `json:"seo_title,omitempty"`
	SEODescription string `json:"seo_description,omitempty"`
}

func registerCategoryTools(server *mcp.Server, client contract.CategoryService, annotations toolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{Name: "cms.category.create", Description: "Create one category and its initial translation. Confirm the category fields with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input categoryCreateInput) (*mcp.CallToolResult, map[string]any, error) {
		if err := validateNamedTranslation(input.Locale, input.Name, input.Slug); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		return toolOutput(client.CreateCategory(writeContext(ctx, req, "cms.category.create", input), contract.CategoryInput{ParentID: input.ParentID, SortOrder: input.SortOrder, Locale: input.Locale, Name: input.Name, Slug: input.Slug, Description: input.Description, SEOTitle: input.SEOTitle, SEODescription: input.SEODescription}))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.category.update", Description: "Update a category's enabled state and sort order. Confirm the target state with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input categoryUpdateInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.CategoryID == 0 {
			return toolError("INVALID_INPUT", "category_id is required"), nil, nil
		}
		return toolOutput(client.UpdateCategory(writeContext(ctx, req, "cms.category.update", input), input.CategoryID, contract.CategoryStateInput{IsEnabled: input.IsEnabled, SortOrder: input.SortOrder}))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.category.move", Description: "Move a category in the hierarchy or change its sort order. Confirm the target parent and order with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input categoryMoveInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.CategoryID == 0 {
			return toolError("INVALID_INPUT", "category_id is required"), nil, nil
		}
		return toolOutput(client.MoveCategory(writeContext(ctx, req, "cms.category.move", input), input.CategoryID, contract.CategoryMoveInput{ParentID: input.ParentID, SortOrder: input.SortOrder}))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.category.upsert_translation", Description: "Create or update one category translation. Confirm the translation fields with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input categoryTranslationInput) (*mcp.CallToolResult, map[string]any, error) {
		if input.CategoryID == 0 {
			return toolError("INVALID_INPUT", "category_id is required"), nil, nil
		}
		if err := validateNamedTranslation(input.Locale, input.Name, input.Slug); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		return toolOutput(client.UpsertCategoryTranslation(writeContext(ctx, req, "cms.category.upsert_translation", input), input.CategoryID, input.Locale, contract.CategoryTranslationInput{Name: input.Name, Slug: input.Slug, Description: input.Description, SEOTitle: input.SEOTitle, SEODescription: input.SEODescription}))
	})
}
