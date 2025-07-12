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

package resources

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/apache/skywalking-mcp/internal/tools"
)

// Embed MQE documentation and examples

//go:embed mqe_examples.json
var mqeExamples string

//go:embed mqe_detailed_syntax.md
var mqeDetailedSyntaxDoc string

//go:embed mqe_ai_prompt.md
var mqeAIPromptDoc string

// AddMQEResources registers MQE-related resources with the MCP server
func AddMQEResources(s *server.MCPServer) {
	// Add detailed MQE syntax documentation as a resource
	s.AddResource(mcp.Resource{
		URI:         "mqe://docs/syntax",
		Name:        "MQE Detailed Syntax Rules",
		Description: "Comprehensive syntax rules and grammar for MQE expressions",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "mqe://docs/syntax",
				MIMEType: "text/markdown",
				Text:     mqeDetailedSyntaxDoc,
			},
		}, nil
	})

	// Add MQE examples as a resource
	s.AddResource(mcp.Resource{
		URI:         "mqe://docs/examples",
		Name:        "MQE Examples",
		Description: "Common MQE expression examples with natural language descriptions",
		MIMEType:    "application/json",
	}, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "mqe://docs/examples",
				MIMEType: "application/json",
				Text:     mqeExamples,
			},
		}, nil
	})

	// Add a dynamic resource that lists available metrics
	s.AddResource(mcp.Resource{
		URI:         "mqe://metrics/available",
		Name:        "Available Metrics",
		Description: "List of all available metrics in the current SkyWalking instance",
		MIMEType:    "application/json",
	}, func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Call list_mqe_metrics to get real-time data
		resp, err := tools.ListMQEMetricsInternal(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to list metrics: %w", err)
		}

		// Format the response
		var formattedData interface{}
		if unmarshalErr := json.Unmarshal([]byte(resp), &formattedData); unmarshalErr != nil {
			return nil, fmt.Errorf("failed to parse metrics data: %w", unmarshalErr)
		}

		// Convert back to JSON with proper formatting
		formattedJSON, err := json.MarshalIndent(formattedData, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to format metrics data: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "mqe://metrics/available",
				MIMEType: "application/json",
				Text:     string(formattedJSON),
			},
		}, nil
	})

	// Add AI understanding guide as a resource
	s.AddResource(mcp.Resource{
		URI:         "mqe://docs/ai_prompt",
		Name:        "MQE AI Understanding Guide",
		Description: "Guide for AI models to understand natural language queries and convert to MQE",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "mqe://docs/ai_prompt",
				MIMEType: "text/markdown",
				Text:     mqeAIPromptDoc,
			},
		}, nil
	})
}
