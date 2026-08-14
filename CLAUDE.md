# CLAUDE.md - AI Assistant Guide for Apache SkyWalking MCP

This file provides guidance for AI assistants working with the Apache SkyWalking MCP codebase.

## Project Overview

Apache SkyWalking MCP — an MCP (Model Context Protocol) server that bridges AI agents with Apache SkyWalking OAP via GraphQL. It exposes SkyWalking's observability data (traces, logs, metrics, topology, alarms, events) as MCP tools and prompts. Binary name: `swmcp`.

## Repository Structure

```
skywalking-mcp/
├── cmd/skywalking-mcp/       # Entry point (cobra/viper CLI, three subcommands)
├── internal/
│   ├── config/               # Config structs for each transport mode
│   ├── swmcp/                # MCP server factory + transport adapters (stdio/sse/streamable)
│   ├── tools/                # MCP tool implementations (16 tools, grouped by domain)
│   └── prompts/              # MCP prompt definitions (10 prompts, three groups)
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

Unit tests exist for selected transport/context behavior. CI runs license checks, lint, and docker build.

## Architecture

### Transport & Context Flow

Three MCP transport modes as cobra subcommands: `stdio`, `sse`, `streamable`.

The SkyWalking OAP URL is resolved in priority order:
- **All transports**: `--sw-url` flag > `http://localhost:12800/graphql`

SSE and HTTP transports always use the configured server URL.

Basic auth is configured via `--sw-username` / `--sw-password` flags. The startup flags support `${ENV_VAR}` syntax to resolve credentials from environment variables (e.g. `--sw-password ${MY_SECRET}`). If a referenced env var is not set, a warning is logged and the credential is treated as empty.

TLS verification is enforced by default. Use `--sw-insecure` to skip verification (development/self-signed certs only).

`skyWalkingContextMiddleware()` injects the OAP URL, insecure flag, and auth into the request context via `WithSkyWalkingURLAndInsecure()` and `WithSkyWalkingAuth()`. It is registered with `AddReceivingMiddleware` and therefore covers every transport. Tools extract the values downstream using `skywalking-cli`'s `contextkey.BaseURL{}`, `contextkey.Insecure{}`, `contextkey.Username{}`, and `contextkey.Password{}`. The middleware deliberately reads only the configuration, never request headers, so a client cannot redirect queries to another OAP.

### CORS / CSRF (`internal/swmcp/cors.go`)

`sse` and `streamable` transports support `--allowed-origins` (comma-separated). When set, requests with an `Origin` header not in the list are rejected with `403 Forbidden`. CORS response headers are set for allowed origins. When the flag is empty (default), all origins are permitted. The middleware is injected via `WithHTTPServer` / `WithStreamableHTTPServer` so the MCP handler is wrapped rather than forked.

### Server Wiring (`internal/swmcp/server.go`)

`newMCPServer()` is the central registration point — it creates the MCP server and calls all `Add*Tools()` and `Add*Prompts()` functions. New capabilities must be registered here.

### Tool Schemas (`internal/tools/result.go`)

Tools are registered with `mcp.AddTool`, which binds JSON arguments to the handler's input struct and infers the input schema from it. `InferSchema[T]()` derives that schema; since the `jsonschema` struct tag can only carry a description, the helpers next to it patch the rest onto the inferred schema:

- `WithRequired` declares required properties explicitly, rather than leaving the contract to depend on `omitempty`
- `WithEnum` constrains a property to a fixed set of values
- `WithDescriptions` sets descriptions a struct tag cannot hold (backticks, newlines, or lines over the length limit)

`ResultText` / `ResultError` build tool results. All tools are marked idempotent.

### Communication with SkyWalking OAP

- **Most tools** use `skywalking-cli` packages (`pkg/graphql/...`) which communicate via GraphQL
- **MQE tools** use direct HTTP calls to the OAP `/graphql` endpoint via `executeGraphQLWithContext()` in `mqe.go`. The HTTP client reads `contextkey.Insecure{}` to configure TLS and validates the URL scheme (`http`/`https` only) before each request.
- **Time handling**: `common.go` provides `BuildDurationWithContext()` and `GetTimeContext()` which fetch the OAP server's time/timezone for accurate duration calculations

### Input Validation (`internal/tools/mqe.go`)

All MQE tool inputs are validated before use:
- `validateMQETextField`: UTF-8, max length, no control characters — applied to all entity fields
- `validateLayerField`: additionally enforces `^[A-Z0-9_]+$` for `layer` / `dest_layer`
- `validateMQEExpression`: UTF-8, max 2048 chars, no control characters, max nesting depth 12
- `validateMetricName`: `^[A-Za-z0-9_.:-]+$` pattern, max 128 chars
- `validateRegexComplexity`: parses the regex AST via `regexp/syntax` and rejects patterns with >50 nodes
- `validateURLScheme` (`common.go`): rejects non-http/https OAP URLs before HTTP requests

## Extending the Server

### Adding a New Tool
1. Create or edit a file in `internal/tools/` (group by domain, e.g. `event.go`)
2. Define the request struct with `json` tags (use `omitempty`) and `jsonschema` tags for descriptions
3. Write the handler with the `mcp.ToolHandlerFor` signature, plus a `xxxTool()` function returning the `*mcp.Tool`
4. Register it with `mcp.AddTool` in the domain's `Add*Tools()` function
5. Register that function in `newMCPServer()` in `server.go`
6. Follow existing tools (e.g. `event.go`) as reference for the pattern

### Adding a New Prompt
1. Add handler in `internal/prompts/` (analysis, trace, or utility group)
2. Register via `s.AddPrompt()` in the corresponding group function in `registry.go`

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
Tool handlers should return `(ResultError(...), nil, nil)` for expected query failures (bad input, OAP errors), not a Go error. Reserve Go errors for truly unexpected failures. Use the `ErrMarshalFailed` constant for JSON marshal errors.

### Comments
Add a comment only when the code is not clear on its own — explain the non-obvious *why* (a workaround, a compatibility constraint, a surprising side effect), not the *what*. Do not write doc comments that merely restate a function/const name. Example worth keeping: `queryTraceV1GQL` omits the `duration` arg because it does not exist on OAP < 10.3.0.

## CI & Merge Policy

Squash-merge only. PRs to `main` require 1 approval and passing `Required` status check (license + lint + docker build). Go 1.26.

### GitHub Actions & the ASF allowed-actions list
This repo is under `apache/*` and is governed by the ASF org-wide allowed-actions policy, enforced on every workflow run:
- Actions in `apache/*`, `actions/*`, and `github/*` are always allowed — no pinning required.
- Every **third-party** action (e.g. `docker/*`) must be pinned to a specific commit **SHA** that is on the ASF approved list. The list **rotates**: old SHAs are pruned as new action versions are approved, so a pin that passed months ago can later be rejected with "action is not allowed". `publish-docker.yaml` only runs on push-to-`main`/release, so a stale pin stays latent until the next publish.
- Before relying on or bumping a third-party action, check the SHA against the approved list: https://github.com/apache/infrastructure-actions/blob/main/approved_patterns.yml (entries are `owner/action@<SHA>`; pick the newest approved SHA of that action). Only if a needed action/SHA is absent does it require a PR to `apache/infrastructure-actions`.

### Commit authorship
Commits produced with AI assistance credit the assistant as a co-author, e.g. a `Co-Authored-By: Claude <...>` trailer, so the contribution is attributed transparently.
