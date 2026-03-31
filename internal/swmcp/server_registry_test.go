// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package swmcp

import (
	"reflect"
	"sort"
	"testing"
	"unsafe"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// These registry tests verify that newMCPServer wires up the expected tools,
// prompts, and resources. Prefer MCPServer public APIs where available. As of
// mcp-go v0.45.0 only tools have a public inventory API, so prompt/resource
// assertions go through the helper layer below. If a future mcp-go release
// exposes prompt/resource listing, replace the reflection helper there rather
// than spreading reflect/unsafe access across tests.

func TestNewMCPServerRegistersExpectedTools(t *testing.T) {
	srv := newMCPServer()

	got := sortedToolNames(srv)
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
	srv := newMCPServer()
	inventory := inspectServerInventory(srv)

	got := sortedPromptNames(inventory.prompts)
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

func TestNewMCPServerRegistersExpectedResources(t *testing.T) {
	srv := newMCPServer()
	inventory := inspectServerInventory(srv)

	resources := inventory.resources
	got := make([]string, 0, len(resources))
	for uri := range resources {
		got = append(got, uri)
	}
	sort.Strings(got)

	want := []string{
		"mqe://docs/ai_prompt",
		"mqe://docs/examples",
		"mqe://docs/syntax",
		"mqe://metrics/available",
	}

	assertStringSlicesEqual(t, got, want)
}

func TestPromptMetadataIncludesExpectedArguments(t *testing.T) {
	srv := newMCPServer()
	prompts := inspectServerInventory(srv).prompts

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

func TestResourceMetadataIncludesExpectedMIMETypes(t *testing.T) {
	srv := newMCPServer()
	resources := inspectServerInventory(srv).resources

	tests := []struct {
		uri      string
		name     string
		mimeType string
	}{
		{uri: "mqe://docs/syntax", name: "MQE Detailed Syntax Rules", mimeType: "text/markdown"},
		{uri: "mqe://docs/examples", name: "MQE Examples", mimeType: "application/json"},
		{uri: "mqe://metrics/available", name: "Available Metrics", mimeType: "application/json"},
		{uri: "mqe://docs/ai_prompt", name: "MQE AI Understanding Guide", mimeType: "text/markdown"},
	}

	for _, tc := range tests {
		t.Run(tc.uri, func(t *testing.T) {
			resource, ok := resources[tc.uri]
			if !ok {
				t.Fatalf("resource %q not registered", tc.uri)
			}
			if resource.Name != tc.name {
				t.Fatalf("resource name = %q, want %q", resource.Name, tc.name)
			}
			if resource.MIMEType != tc.mimeType {
				t.Fatalf("resource MIME type = %q, want %q", resource.MIMEType, tc.mimeType)
			}
		})
	}
}

func TestToolMetadataIncludesExpectedDescriptionsAndSchemas(t *testing.T) {
	srv := newMCPServer()
	tools := toolMap(srv)

	tests := []struct {
		name             string
		expectDesc       bool
		expectProperties []string
	}{
		{name: "query_traces", expectDesc: true, expectProperties: []string{"service_id", "trace_id", "view"}},
		{name: "execute_mqe_expression", expectDesc: true, expectProperties: []string{"expression", "service_name", "debug"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tool, ok := tools[tc.name]
			if !ok {
				t.Fatalf("tool %q not registered", tc.name)
			}
			if tc.expectDesc && tool.Description == "" {
				t.Fatalf("tool %q description is empty", tc.name)
			}
			properties := tool.InputSchema.Properties
			for _, property := range tc.expectProperties {
				if _, ok := properties[property]; !ok {
					t.Fatalf("tool %q missing input schema property %q", tc.name, property)
				}
			}
		})
	}
}

func toolMap(srv *server.MCPServer) map[string]mcp.Tool {
	serverTools := srv.ListTools()
	result := make(map[string]mcp.Tool, len(serverTools))
	for name, tool := range serverTools {
		result[name] = tool.Tool
	}

	return result
}

func sortedToolNames(srv *server.MCPServer) []string {
	tools := toolMap(srv)
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedPromptNames(prompts map[string]mcp.Prompt) []string {
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

func readPrivateField(v reflect.Value) reflect.Value {
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func testedServerValue(srv *server.MCPServer) reflect.Value {
	return reflect.ValueOf(srv).Elem()
}

func mustReadServerField(srv reflect.Value, fieldName string) reflect.Value {
	field := srv.FieldByName(fieldName)
	if !field.IsValid() {
		panic("mcp-go MCPServer no longer has field " + fieldName)
	}
	return readPrivateField(field)
}

type serverInventory struct {
	prompts   map[string]mcp.Prompt
	resources map[string]mcp.Resource
}

func inspectServerInventory(srv *server.MCPServer) serverInventory {
	serverValue := testedServerValue(srv)
	return serverInventory{
		prompts:   readPromptMap(serverValue),
		resources: readResourceMap(serverValue),
	}
}

func readPromptMap(serverValue reflect.Value) map[string]mcp.Prompt {
	serverPrompts := mustReadServerField(serverValue, "prompts")
	result := make(map[string]mcp.Prompt, serverPrompts.Len())

	iter := serverPrompts.MapRange()
	for iter.Next() {
		result[iter.Key().String()] = copyReflectValue(iter.Value()).Interface().(mcp.Prompt)
	}

	return result
}

func readResourceMap(serverValue reflect.Value) map[string]mcp.Resource {
	serverResources := mustReadServerField(serverValue, "resources")
	result := make(map[string]mcp.Resource, serverResources.Len())

	iter := serverResources.MapRange()
	for iter.Next() {
		resourceField := copyReflectValue(iter.Value()).FieldByName("resource")
		result[iter.Key().String()] = readPrivateField(resourceField).Interface().(mcp.Resource)
	}

	return result
}

func copyReflectValue(v reflect.Value) reflect.Value {
	cloned := reflect.New(v.Type()).Elem()
	cloned.Set(v)
	return cloned
}
