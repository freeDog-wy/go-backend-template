package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type localeCreateInput struct {
	Code      string `json:"code" jsonschema:"BCP 47-like locale code, such as en-US"`
	Name      string `json:"name" jsonschema:"human-readable locale name"`
	IsEnabled bool   `json:"is_enabled" jsonschema:"whether this locale is immediately visible publicly"`
	SortOrder int    `json:"sort_order,omitempty" jsonschema:"display order"`
}

type localeUpdateInput struct {
	Code         string `json:"code" jsonschema:"existing locale code"`
	Name         string `json:"name" jsonschema:"human-readable locale name"`
	IsEnabled    bool   `json:"is_enabled" jsonschema:"whether this locale is visible publicly"`
	SortOrder    int    `json:"sort_order,omitempty" jsonschema:"display order"`
	SetAsDefault bool   `json:"set_as_default,omitempty" jsonschema:"make this enabled locale the default locale"`
}

func registerLocaleTools(server *mcp.Server, client contract.LocaleService, annotations toolAnnotations) {
	mcp.AddTool(server, &mcp.Tool{Name: "cms.locale.create", Description: "Create one CMS locale. Disabled locales remain unavailable publicly until enabled. Confirm the locale code, name, and initial enabled state with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input localeCreateInput) (*mcp.CallToolResult, map[string]any, error) {
		if err := validateLocaleInput(input.Code, input.Name); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		return toolOutput(client.CreateLocale(writeContext(ctx, req, "cms.locale.create", input), contract.LocaleCreateInput{Code: input.Code, Name: input.Name, IsEnabled: input.IsEnabled, SortOrder: input.SortOrder}))
	})
	mcp.AddTool(server, &mcp.Tool{Name: "cms.locale.update", Description: "Update a CMS locale's name, public enabled state, display order, or default status. Confirm the full target state with the user before calling.", Annotations: annotations.write}, func(ctx context.Context, req *mcp.CallToolRequest, input localeUpdateInput) (*mcp.CallToolResult, map[string]any, error) {
		if err := validateLocaleInput(input.Code, input.Name); err != nil {
			return toolError("INVALID_INPUT", err.Error()), nil, nil
		}
		return toolOutput(client.UpdateLocale(writeContext(ctx, req, "cms.locale.update", input), input.Code, contract.LocaleUpdateInput{Name: input.Name, IsEnabled: input.IsEnabled, SortOrder: input.SortOrder, IsDefault: input.SetAsDefault}))
	})
}

func validateLocaleInput(code, name string) error {
	code = strings.TrimSpace(code)
	if len(code) < 2 || len(code) > 35 {
		return fmt.Errorf("locale code must contain 2 to 35 characters")
	}
	for _, r := range code {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return fmt.Errorf("locale code may contain only letters, digits, and hyphens")
		}
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("locale name is required")
	}
	return nil
}

func validateNamedTranslation(locale, name, slug string) error {
	if strings.TrimSpace(locale) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(slug) == "" {
		return fmt.Errorf("locale, name, and slug are required")
	}
	return nil
}
