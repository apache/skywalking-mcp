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
	"reflect"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// These registry tests verify that newMCPServer wires up the expected tools and
// prompts. They list them over a real in-memory session, so the assertions run
// through the same path a client would.

// newTestSession connects a client to newMCPServer over an in-memory transport.
func newTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	serverSession, err := newMCPServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil).
		Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func TestNewMCPServerRegistersExpectedTools(t *testing.T) {
	res, err := newTestSession(t).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	sort.Strings(got)

	want := []string{
		"execute_mqe_expression",
		"get_mqe_metric_type",
		"list_endpoints",
		"list_instances",
		"list_layers",
		"list_mqe_metrics",
		"list_processes",
		"list_services",
		"query_alarms",
		"query_endpoints_topology",
		"query_events",
		"query_instances_topology",
		"query_logs",
		"query_processes_topology",
		"query_services_topology",
		"query_traces",
	}

	assertStringSlicesEqual(t, got, want)
}

func TestNewMCPServerRegistersExpectedPrompts(t *testing.T) {
	got := sortedPromptNames(t)
	want := []string{
		"analyze-logs",
		"analyze-performance",
		"build-mqe-query",
		"compare-services",
		"explore-metrics",
		"explore-service-topology",
		"generate_duration",
		"investigate-traces",
		"top-services",
		"trace-deep-dive",
	}

	assertStringSlicesEqual(t, got, want)
}

func TestPromptMetadataIncludesExpectedArguments(t *testing.T) {
	prompts := promptMap(t)

	prompt, ok := prompts["generate_duration"]
	if !ok {
		t.Fatal("generate_duration prompt not registered")
	}
	if prompt.Description == "" {
		t.Fatal("generate_duration prompt description is empty")
	}
	if len(prompt.Arguments) != 1 {
		t.Fatalf("generate_duration prompt arguments = %d, want 1", len(prompt.Arguments))
	}
	if prompt.Arguments[0].Name != "time_range" || !prompt.Arguments[0].Required {
		t.Fatalf("unexpected generate_duration argument: %+v", prompt.Arguments[0])
	}

	tracePrompt, ok := prompts["trace-deep-dive"]
	if !ok {
		t.Fatal("trace-deep-dive prompt not registered")
	}
	if len(tracePrompt.Arguments) != 2 {
		t.Fatalf("trace-deep-dive prompt arguments = %d, want 2", len(tracePrompt.Arguments))
	}
	if tracePrompt.Arguments[0].Name != "trace_id" || !tracePrompt.Arguments[0].Required {
		t.Fatalf("unexpected first trace-deep-dive argument: %+v", tracePrompt.Arguments[0])
	}
}

func TestToolMetadataIncludesExpectedDescriptionsAndSchemas(t *testing.T) {
	res, err := newTestSession(t).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	tools := make(map[string]*mcp.Tool, len(res.Tools))
	for _, tool := range res.Tools {
		tools[tool.Name] = tool
	}

	tests := []struct {
		name             string
		expectProperties []string
	}{
		{name: "query_traces", expectProperties: []string{"service_id", "trace_id", "view"}},
		{name: "execute_mqe_expression", expectProperties: []string{"expression", "service_name", "debug"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, ok := tools[tc.name]
			if !ok {
				t.Fatalf("tool %q not registered", tc.name)
			}
			if tool.Description == "" {
				t.Fatalf("tool %q description is empty", tc.name)
			}

			// Over the wire the schema is plain JSON, not a *jsonschema.Schema.
			schema, ok := tool.InputSchema.(map[string]any)
			if !ok {
				t.Fatalf("tool %q input schema type = %T", tc.name, tool.InputSchema)
			}
			properties, ok := schema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("tool %q input schema has no properties", tc.name)
			}
			for _, property := range tc.expectProperties {
				if _, ok := properties[property]; !ok {
					t.Fatalf("tool %q missing input schema property %q", tc.name, property)
				}
			}
		})
	}
}

func promptMap(t *testing.T) map[string]*mcp.Prompt {
	t.Helper()

	res, err := newTestSession(t).ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatalf("list prompts: %v", err)
	}

	prompts := make(map[string]*mcp.Prompt, len(res.Prompts))
	for _, prompt := range res.Prompts {
		prompts[prompt.Name] = prompt
	}
	return prompts
}

func sortedPromptNames(t *testing.T) []string {
	t.Helper()

	prompts := promptMap(t)
	names := make([]string, 0, len(prompts))
	for name := range prompts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func assertStringSlicesEqual(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("values mismatch:\n got: %v\nwant: %v", got, want)
	}
}
