package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mcpauth "github.com/freeDog-wy/go-backend-template/internal/app/mcp/auth"
	mcpconfig "github.com/freeDog-wy/go-backend-template/internal/app/mcp/config"
	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/gsc"
	mcpserver "github.com/freeDog-wy/go-backend-template/internal/app/mcp/server"
	"github.com/freeDog-wy/go-backend-template/internal/app/pkg/cmsclient"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Optional path to the MCP configuration file")
	flag.Parse()

	cfg, err := mcpconfig.Load(configPath)
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("validate MCP configuration: %v", err)
	}

	httpClient := &http.Client{Timeout: time.Duration(cfg.RequestTimeoutSeconds) * time.Second}
	provider, err := mcpauth.New(cfg.CMSBaseURL, cfg.ClientID, cfg.ClientSecret, httpClient, cfg.AllowInsecureHTTP)
	if err != nil {
		log.Fatalf("initialize service token provider: %v", err)
	}
	client, err := cmsclient.NewAdmin(cfg.CMSBaseURL, httpClient, cmsclient.BearerAuthorizer(provider), cfg.AllowInsecureHTTP)
	if err != nil {
		log.Fatalf("initialize CMS client: %v", err)
	}
	var searchConsole *gsc.Client
	if cfg.GSCEnabled {
		searchConsole, err = gsc.New(cfg.GSCProperty, cfg.GSCServiceAccountFile, httpClient)
		if err != nil {
			log.Fatalf("initialize Google Search Console client: %v", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	deps := mcpserver.Dependencies{
		Site:          client,
		Locales:       client,
		Articles:      client,
		Categories:    client,
		Tags:          client,
		SearchConsole: searchConsole,
		ContentRoot:   cfg.ContentRoot,
	}
	mcpLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})).With("component", "cms-mcp")
	if err := mcpserver.New(deps, mcpLogger).Run(ctx, &mcp.StdioTransport{}); err != nil && ctx.Err() == nil {
		log.Printf("MCP server stopped: %v", err)
	}
}
