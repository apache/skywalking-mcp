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
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-cli/pkg/contextkey"

	"github.com/apache/skywalking-mcp/internal/config"
	"github.com/apache/skywalking-mcp/internal/prompts"
	"github.com/apache/skywalking-mcp/internal/resources"
	"github.com/apache/skywalking-mcp/internal/tools"
)

// newMCPServer creates a new MCP server with all tools, resources, and prompts registered.
func newMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"skywalking-mcp", "0.1.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)
	tools.AddTraceTools(s)
	tools.AddLogTools(s)
	tools.AddMQETools(s)
	tools.AddMetadataTools(s)
	tools.AddEventTools(s)
	tools.AddAlarmTools(s)
	tools.AddTopologyTools(s)
	resources.AddMQEResources(s)
	prompts.AddSkyWalkingPrompts(s)
	return s
}

func initLogger(logFilePath string) (*logrus.Logger, error) {
	if logFilePath == "" {
		return logrus.New(), nil
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logrusLogger := logrus.New()
	logrusLogger.SetFormatter(&logrus.TextFormatter{})
	logrusLogger.SetLevel(logrus.DebugLevel)
	logrusLogger.SetOutput(file)

	return logrusLogger, nil
}

// WithSkyWalkingURLAndInsecure adds SkyWalking URL and insecure flag to the context
// This ensures all downstream requests will have contextkey.BaseURL{} and contextkey.Insecure{}
func WithSkyWalkingURLAndInsecure(ctx context.Context, url string, insecure bool) context.Context {
	ctx = context.WithValue(ctx, contextkey.BaseURL{}, url)
	ctx = context.WithValue(ctx, contextkey.Insecure{}, insecure)
	return ctx
}

// configuredSkyWalkingURL returns the configured SkyWalking OAP URL.
// The value is sourced from the CLI/config binding for `--sw-url`,
// falling back to the built-in default when unset.
func configuredSkyWalkingURL() string {
	urlStr := viper.GetString("url")
	if urlStr == "" {
		urlStr = config.DefaultSWURL
	}
	return tools.FinalizeURL(urlStr)
}

// urlFromHeaders extracts URL for a request.
// URL is sourced from Header > configured value > Default.
func urlFromHeaders(req *http.Request) string {
	urlStr := req.Header.Get("SW-URL")
	if urlStr == "" {
		return configuredSkyWalkingURL()
	}

	return tools.FinalizeURL(urlStr)
}

// WithSkyWalkingContextFromConfig injects the SkyWalking URL and insecure
// settings from global configuration into the context.
var WithSkyWalkingContextFromConfig server.StdioContextFunc = func(ctx context.Context) context.Context {
	return WithSkyWalkingURLAndInsecure(ctx, configuredSkyWalkingURL(), false)
}

// withSkyWalkingContextFromRequest is the shared logic for enriching context from an http.Request.
func withSkyWalkingContextFromRequest(ctx context.Context, req *http.Request) context.Context {
	urlStr := urlFromHeaders(req)
	return WithSkyWalkingURLAndInsecure(ctx, urlStr, false)
}

// EnhanceStdioContextFunc returns a StdioContextFunc that enriches the context
// with SkyWalking settings from the global configuration.
func EnhanceStdioContextFunc() server.StdioContextFunc {
	return WithSkyWalkingContextFromConfig
}

// EnhanceSSEContextFunc returns a SSEContextFunc that enriches the context
// with SkyWalking settings from SSE request headers.
func EnhanceSSEContextFunc() server.SSEContextFunc {
	return withSkyWalkingContextFromRequest
}

// EnhanceHTTPContextFunc returns a HTTPContextFunc that enriches the context
// with SkyWalking settings from HTTP request headers.
func EnhanceHTTPContextFunc() server.HTTPContextFunc {
	return withSkyWalkingContextFromRequest
}
