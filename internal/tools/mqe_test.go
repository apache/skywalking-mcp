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
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apache/skywalking-cli/pkg/contextkey"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestValidateMQEExpressionRequestRejectsDeeplyNestedExpression(t *testing.T) {
	req := &MQEExpressionRequest{
		Expression: strings.Repeat("(", maxMQEExpressionDepth+1) + "service_cpm" + strings.Repeat(")", maxMQEExpressionDepth+1),
	}

	err := validateMQEExpressionRequest(req)
	if err == nil || !strings.Contains(err.Error(), "maximum nesting depth") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMQEMetricsListRequestRejectsInvalidRegex(t *testing.T) {
	err := validateMQEMetricsListRequest(&MQEMetricsListRequest{Regex: "("})
	if err == nil || err.Error() != "regex is invalid" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMetricNameRejectsInvalidCharacters(t *testing.T) {
	err := validateMetricName("service cpm")
	if err == nil || err.Error() != "metric_name contains invalid characters" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteMQEExpressionRejectsOverlongEntityField(t *testing.T) {
	req := &MQEExpressionRequest{
		Expression:  "service_cpm",
		ServiceName: strings.Repeat("a", maxMQEEntityFieldLen+1),
	}

	result, err := executeMQEExpression(context.Background(), req)
	if err != nil {
		t.Fatalf("executeMQEExpression returned error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error result")
	}
	assertToolResultContains(t, result, "service_name exceeds maximum length")
}

func TestExecuteGraphQLWithContextSanitizesHTTPErrorBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "sensitive backend details", http.StatusBadGateway)
	}))
	defer ts.Close()

	ctx := context.WithValue(context.Background(), contextkey.BaseURL{}, ts.URL)
	_, err := executeGraphQLWithContext(ctx, "query { ping }", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTTP status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "sensitive backend details") {
		t.Fatalf("backend body leaked in error: %v", err)
	}
}

func TestExecuteGraphQLWithContextSanitizesGraphQLErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"database stack trace"}]}`))
	}))
	defer ts.Close()

	ctx := context.WithValue(context.Background(), contextkey.BaseURL{}, ts.URL)
	_, err := executeGraphQLWithContext(ctx, "query { ping }", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "GraphQL query failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertToolResultContains(t *testing.T, result *mcp.CallToolResult, want string) {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("tool result had no content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	if !strings.Contains(text.Text, want) {
		t.Fatalf("tool result text %q does not contain %q", text.Text, want)
	}
}
