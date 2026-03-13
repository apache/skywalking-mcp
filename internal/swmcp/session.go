// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package swmcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/apache/skywalking-mcp/internal/tools"
)

// sessionKey is the context key for looking up the session store.
type sessionKey struct{}

// Session holds per-session SkyWalking connection configuration.
type Session struct {
	mu       sync.RWMutex
	url      string
	username string
	password string
}

// SetConnection updates the session's connection parameters.
func (s *Session) SetConnection(url, username, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.url = url
	s.username = username
	s.password = password
}

// URL returns the session's configured URL, or empty if not set.
func (s *Session) URL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.url
}

// Username returns the session's configured username.
func (s *Session) Username() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.username
}

// Password returns the session's configured password.
func (s *Session) Password() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.password
}

// SessionFromContext retrieves the session from the context, or nil if not present.
func SessionFromContext(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionKey{}).(*Session)
	return s
}

// WithSession attaches a session to the context.
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// SetSkyWalkingURLRequest represents the request for the set_skywalking_url tool.
type SetSkyWalkingURLRequest struct {
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func setSkyWalkingURL(ctx context.Context, req *SetSkyWalkingURLRequest) (*mcp.CallToolResult, error) {
	if req.URL == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	session := SessionFromContext(ctx)
	if session == nil {
		return mcp.NewToolResultError("session not available"), nil
	}

	finalURL := tools.FinalizeURL(req.URL)
	session.SetConnection(finalURL, resolveEnvVar(req.Username), resolveEnvVar(req.Password))

	msg := fmt.Sprintf("SkyWalking URL set to %s", finalURL)
	if req.Username != "" {
		msg += " with basic auth credentials"
	}
	return mcp.NewToolResultText(msg), nil
}

// AddSessionTools registers session management tools with the MCP server.
func AddSessionTools(s *server.MCPServer) {
	tool := tools.NewTool(
		"set_skywalking_url",
		`Set the SkyWalking OAP server URL and optional basic auth credentials for this session.

This tool configures the connection to SkyWalking OAP for all subsequent tool calls in the current session.
The URL and credentials persist for the lifetime of the session.

Priority: session URL (set by this tool) > --sw-url flag > default (http://localhost:12800/graphql)

Credentials support raw values or environment variable references using ${ENV_VAR} syntax.

Examples:
- {"url": "http://demo.skywalking.apache.org:12800"}: Connect without auth
- {"url": "http://oap.internal:12800", "username": "admin", "password": "admin"}: Connect with basic auth
- {"url": "https://skywalking.example.com:443", "username": "${SW_USER}", "password": "${SW_PASS}"}: Auth via env vars`,
		setSkyWalkingURL,
		mcp.WithString("url", mcp.Required(),
			mcp.Description("SkyWalking OAP server URL (required). Example: http://localhost:12800")),
		mcp.WithString("username",
			mcp.Description("Username for basic auth (optional). Supports ${ENV_VAR} syntax.")),
		mcp.WithString("password",
			mcp.Description("Password for basic auth (optional). Supports ${ENV_VAR} syntax.")),
	)
	tool.Register(s)
}
