# CLAUDE.md - AI Assistant Guide for Apache SkyWalking MCP

This file provides guidance for AI assistants working with the Apache SkyWalking MCP codebase.

## Project Overview

Apache SkyWalking MCP — an MCP (Model Context Protocol) server that bridges AI agents with Apache SkyWalking OAP via GraphQL. It exposes SkyWalking's observability data (traces, logs, metrics, topology, alarms, events) as MCP tools, prompts, and resources. Binary name: `swmcp`.

## Repository Structure

```
skywalking-mcp/
├── cmd/skywalking-mcp/       # Entry point (cobra/viper CLI, three subcommands)
├── internal/
│   ├── config/               # Config structs for each transport mode
│   ├── swmcp/                # MCP server factory + transport adapters (stdio/sse/streamable)
│   ├── tools/                # MCP tool implementations (16 tools, grouped by domain)
│   ├── prompts/              # MCP prompt definitions (10 prompts, three groups)
│   └── resources/            # MCP resources (embedded MQE docs + dynamic metrics)
└── dist/                     # Distribution license files
```

## Build & Development Commands

```bash
make build            # Build binary to bin/swmcp
make lint             # Run golangci-lint (22 linters)
make fix-lint         # Auto-fix lint issues
make license-header   # Check Apache 2.0 license headers
make fix-license      # Fix license headers and dependency licenses
make build-image      # Build Docker image skywalking-mcp:latest
make clean            # Remove build artifacts
```

No unit tests exist yet. CI runs license checks, lint, and docker build.

## Architecture

### Transport & Context Flow

Three MCP transport modes as cobra subcommands: `stdio`, `sse`, `streamable`.

The SkyWalking OAP URL is resolved in priority order:
`--sw-url` flag > `SW_URL` env > `SW-URL` HTTP header > `http://localhost:12800/graphql`

Each transport injects the OAP URL into the request context via `WithSkyWalkingURLAndInsecure()`. Tools extract it downstream using `skywalking-cli`'s `contextkey.BaseURL{}`.

### Server Wiring (`internal/swmcp/server.go`)

`newMCPServer()` is the central registration point — it creates the MCP server and calls all `Add*Tools()`, `Add*Resources()`, and `Add*Prompts()` functions. New capabilities must be registered here.

### Generic Tool Framework (`internal/tools/tools.go`)

`Tool[T, R]` is a typed generic wrapper over MCP's untyped interface. `ConvertTool()` bridges typed handlers into MCP by auto-binding JSON arguments to `T` and marshaling `R` back to JSON. If `R` is already `*mcp.CallToolResult`, it passes through directly. All tools are marked idempotent by default.

### Communication with SkyWalking OAP

- **Most tools** use `skywalking-cli` packages (`pkg/graphql/...`) which communicate via GraphQL
- **MQE tools** use direct HTTP calls to the OAP `/graphql` endpoint
- **Time handling**: `common.go` provides `BuildDurationWithContext()` and `GetTimeContext()` which fetch the OAP server's time/timezone for accurate duration calculations

## Extending the Server

### Adding a New Tool
1. Create or edit a file in `internal/tools/` (group by domain, e.g. `event.go`)
2. Define request struct with `json` tags, write handler using `NewTool()`, create `Add*Tools()` function
3. Register in `newMCPServer()` in `server.go`
4. Follow existing tools (e.g. `event.go`) as reference for the pattern

### Adding a New Prompt
1. Add handler in `internal/prompts/` (analysis, trace, or utility group)
2. Register via `s.AddPrompt()` in the corresponding group function in `registry.go`

### Adding a New Resource
1. For static content: embed files with `//go:embed` in `internal/resources/`
2. For dynamic content: call internal tool functions in the resource handler
3. Register via `s.AddResource()` in `AddMQEResources()` or a new registration function

## Code Conventions

### License Header
All `.go` files must have the Apache 2.0 license header (17-line block). Run `make fix-license` to auto-fix.

### Lint Rules (`.golangci.yml`)
- Max function length: 100 lines / 50 statements
- Cyclomatic complexity: 15
- Line length: 150 chars
- Imports: `goimports` with local prefix `github.com/apache/skywalking-mcp`
- Import order: stdlib, third-party, blank line, local packages
- 22 linters enabled including gosec, errcheck, dupl, gocritic

### Error Handling in Tools
Tool handlers should return `(mcp.NewToolResultError(...), nil)` for expected query failures (bad input, OAP errors), not `(nil, err)`. Reserve Go errors for truly unexpected failures. Use the `ErrMarshalFailed` constant for JSON marshal errors.

## CI & Merge Policy

Squash-merge only. PRs to `main` require 1 approval and passing `Required` status check (license + lint + docker build). Go 1.25.