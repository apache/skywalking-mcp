package swmcp

import (
	"reflect"
	"sort"
	"testing"
	"unsafe"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestNewMCPServerRegistersExpectedToolsForStdio(t *testing.T) {
	srv := newMCPServer(true)

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
		"set_skywalking_url",
	}

	assertStringSlicesEqual(t, got, want)
}

func TestNewMCPServerDoesNotRegisterSessionToolForNetworkTransports(t *testing.T) {
	srv := newMCPServer(false)

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
	srv := newMCPServer(false)

	got := sortedPromptNames(srv)
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
	srv := newMCPServer(false)

	resources := resourceMap(srv)
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
	srv := newMCPServer(false)
	prompts := promptMap(srv)

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
	srv := newMCPServer(false)
	resources := resourceMap(srv)

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
	srv := newMCPServer(true)
	tools := toolMap(srv)

	tests := []struct {
		name             string
		expectDesc       bool
		expectProperties []string
	}{
		{name: "set_skywalking_url", expectDesc: true, expectProperties: []string{"url", "username", "password"}},
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
	serverTools := readPrivateField(reflect.ValueOf(srv).Elem().FieldByName("tools"))
	result := make(map[string]mcp.Tool, serverTools.Len())

	iter := serverTools.MapRange()
	for iter.Next() {
		name := iter.Key().String()
		toolValue := copyReflectValue(iter.Value())
		result[name] = toolValue.FieldByName("Tool").Interface().(mcp.Tool)
	}

	return result
}

func promptMap(srv *server.MCPServer) map[string]mcp.Prompt {
	serverPrompts := readPrivateField(reflect.ValueOf(srv).Elem().FieldByName("prompts"))
	result := make(map[string]mcp.Prompt, serverPrompts.Len())

	iter := serverPrompts.MapRange()
	for iter.Next() {
		result[iter.Key().String()] = copyReflectValue(iter.Value()).Interface().(mcp.Prompt)
	}

	return result
}

func resourceMap(srv *server.MCPServer) map[string]mcp.Resource {
	serverResources := readPrivateField(reflect.ValueOf(srv).Elem().FieldByName("resources"))
	result := make(map[string]mcp.Resource, serverResources.Len())

	iter := serverResources.MapRange()
	for iter.Next() {
		resourceField := copyReflectValue(iter.Value()).FieldByName("resource")
		result[iter.Key().String()] = readPrivateField(resourceField).Interface().(mcp.Resource)
	}

	return result
}

func sortedToolNames(srv *server.MCPServer) []string {
	names := make([]string, 0, len(toolMap(srv)))
	for name := range toolMap(srv) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedPromptNames(srv *server.MCPServer) []string {
	names := make([]string, 0, len(promptMap(srv)))
	for name := range promptMap(srv) {
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

func copyReflectValue(v reflect.Value) reflect.Value {
	copy := reflect.New(v.Type()).Elem()
	copy.Set(v)
	return copy
}
