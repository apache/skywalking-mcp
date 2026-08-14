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

package tools

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"regexp/syntax"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/apache/skywalking-cli/pkg/contextkey"
)

// AddMQETools registers MQE-related tools with the MCP server
func AddMQETools(s *mcp.Server) {
	mcp.AddTool(s, mqeExpressionTool(), executeMQEExpression)
	mcp.AddTool(s, mqeMetricsListTool(), listMQEMetrics)
	mcp.AddTool(s, mqeMetricsTypeTool(), getMQEMetricsType)
}

const (
	maxMQEExpressionLength = 2048
	maxMQEExpressionDepth  = 12
	maxMQEEntityFieldLen   = 256
	maxMQERegexLength      = 256
	maxMetricNameLength    = 128
)

var metricNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)

// layerPattern restricts layer values to the SkyWalking enum format (e.g. GENERAL, K8S_SERVICE).
var layerPattern = regexp.MustCompile(`^[A-Z0-9_]+$`)

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// getContextString safely extracts a string value from context.
func getContextString(ctx context.Context, key any) string {
	if v, ok := ctx.Value(key).(string); ok {
		return v
	}
	return ""
}

// getContextBool safely extracts a bool value from context.
func getContextBool(ctx context.Context, key any) bool {
	if v, ok := ctx.Value(key).(bool); ok {
		return v
	}
	return false
}

// executeGraphQLWithContext executes a GraphQL query using URL and auth from context.
func executeGraphQLWithContext(ctx context.Context, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	rawURL := getContextString(ctx, contextkey.BaseURL{})
	normalizedURL, err := NormalizeOAPURL(rawURL)
	if err != nil {
		return nil, err
	}

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", normalizedURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add basic auth from context if present
	username := getContextString(ctx, contextkey.Username{})
	password := getContextString(ctx, contextkey.Password{})
	if username != "" && password != "" {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
		req.Header.Set("Authorization", auth)
	}

	insecure := getContextBool(ctx, contextkey.Insecure{})
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec // controlled by --sw-insecure operator flag
	client := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphQL request failed with HTTP status %d", resp.StatusCode)
	}

	var graphqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL query failed")
	}

	return &graphqlResp, nil
}

// MQEExpressionRequest represents a request to execute MQE expression
type MQEExpressionRequest struct {
	Expression              string `json:"expression,omitempty"`
	ServiceName             string `json:"service_name,omitempty"`
	Layer                   string `json:"layer,omitempty"`
	ServiceInstanceName     string `json:"service_instance_name,omitempty"`
	EndpointName            string `json:"endpoint_name,omitempty"`
	ProcessName             string `json:"process_name,omitempty"`
	Normal                  *bool  `json:"normal,omitempty"`
	DestServiceName         string `json:"dest_service_name,omitempty"`
	DestLayer               string `json:"dest_layer,omitempty"`
	DestServiceInstanceName string `json:"dest_service_instance_name,omitempty"`
	DestEndpointName        string `json:"dest_endpoint_name,omitempty"`
	DestProcessName         string `json:"dest_process_name,omitempty"`
	DestNormal              *bool  `json:"dest_normal,omitempty"`
	Start                   string `json:"start,omitempty"`
	End                     string `json:"end,omitempty"`
	Step                    string `json:"step,omitempty"`
	Cold                    bool   `json:"cold,omitempty"`
	Debug                   bool   `json:"debug,omitempty"`
	DumpDBRsp               bool   `json:"dump_db_rsp,omitempty"`
}

// MQEMetricsListRequest represents a request to list available metrics
type MQEMetricsListRequest struct {
	Regex string `json:"regex,omitempty"`
}

// MQEMetricsTypeRequest represents a request to get metric type
type MQEMetricsTypeRequest struct {
	MetricName string `json:"metric_name,omitempty"`
}

// ListServicesRequest represents a request to list services
type ListServicesRequest struct {
	Layer string `json:"layer,omitempty" jsonschema:"The layer to list services for. Use list_layers to get available layer names."`
}

// getServiceInfo queries service information using the specified layer
func getServiceInfo(ctx context.Context, serviceName, layer string) bool {
	if serviceName == "" {
		return false
	}

	if layer == "" {
		layer = "GENERAL"
	}

	normal, err := getServiceByName(ctx, serviceName, layer)
	if err != nil {
		return true
	}
	if normal != nil {
		return *normal
	}

	return true
}

// getServiceByName tries to get service info directly by name in specified layer
func getServiceByName(ctx context.Context, serviceName, layer string) (*bool, error) {
	serviceID, err := findServiceID(ctx, serviceName, layer)
	if err != nil {
		return nil, fmt.Errorf("service not found in layer %s: %s", layer, serviceName)
	}
	if serviceID == "" {
		return nil, fmt.Errorf("service not found in layer %s: %s", layer, serviceName)
	}

	query := `
		query getService($serviceId: String!) {
			service: getService(serviceId: $serviceId) {
				id
				name
				normal
				layers
			}
		}
	`

	variables := map[string]interface{}{
		"serviceId": serviceID,
	}

	result, err := executeGraphQLWithContext(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to get service details: %w", err)
	}

	if data, ok := result.Data.(map[string]interface{}); ok {
		if service, ok := data["service"].(map[string]interface{}); ok {
			if normal, ok := service["normal"].(bool); ok {
				return &normal, nil
			}
		}
	}

	return nil, fmt.Errorf("invalid service data returned for: %s", serviceName)
}

// findServiceID finds service ID by name in a specific layer
func findServiceID(ctx context.Context, serviceName, layer string) (string, error) {
	query := `
		query getServices($layer: String!) {
			services: listServices(layer: $layer) {
				id
				name
			}
		}
	`

	variables := map[string]interface{}{
		"layer": layer,
	}

	result, err := executeGraphQLWithContext(ctx, query, variables)
	if err != nil {
		return "", err
	}

	if data, ok := result.Data.(map[string]interface{}); ok {
		if services, ok := data["services"].([]interface{}); ok {
			for _, s := range services {
				svc, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				if svc["name"] == serviceName {
					if id, ok := svc["id"].(string); ok {
						return id, nil
					}
				}
			}
		}
	}

	return "", nil
}

// buildMQEEntity builds the entity from request parameters
func buildMQEEntity(ctx context.Context, req *MQEExpressionRequest) map[string]interface{} {
	entity := make(map[string]interface{})

	// Define a mapping of field names to their corresponding values
	fields := map[string]interface{}{
		"serviceName":             req.ServiceName,
		"serviceInstanceName":     req.ServiceInstanceName,
		"endpointName":            req.EndpointName,
		"processName":             req.ProcessName,
		"destServiceName":         req.DestServiceName,
		"destServiceInstanceName": req.DestServiceInstanceName,
		"destEndpointName":        req.DestEndpointName,
		"destProcessName":         req.DestProcessName,
	}

	// Populate the entity map based on the mapping
	for key, value := range fields {
		if strValue, ok := value.(string); ok && strValue != "" {
			entity[key] = strValue
		}
	}

	// Handle special cases
	if req.ServiceName != "" {
		if req.Normal == nil {
			normal := getServiceInfo(ctx, req.ServiceName, req.Layer)
			entity["normal"] = normal
		} else {
			entity["normal"] = *req.Normal
		}
	} else if req.Normal != nil {
		entity["normal"] = *req.Normal
	}

	if req.DestNormal != nil {
		entity["destNormal"] = *req.DestNormal
	}

	return entity
}

// executeMQEExpression executes MQE expression query
func executeMQEExpression(ctx context.Context, _ *mcp.CallToolRequest, req MQEExpressionRequest) (*mcp.CallToolResult, any, error) {
	if req.Expression == "" {
		return ResultError("expression is required"), nil, nil
	}
	if err := validateMQEExpressionRequest(&req); err != nil {
		return ResultError(err.Error()), nil, nil
	}

	entity := buildMQEEntity(ctx, &req)
	timeCtx := GetTimeContext(ctx)

	duration := BuildDurationWithContext(req.Start, req.End, req.Step, req.Cold, DefaultDuration, timeCtx)

	// GraphQL query for MQE expression
	query := `
		query execExpression($expression: String!, $entity: Entity!, $duration: Duration!, $debug: Boolean, $dumpDBRsp: Boolean) {
			execExpression(expression: $expression, entity: $entity, duration: $duration, debug: $debug, dumpDBRsp: $dumpDBRsp) {
				type
				error
				results {
					metric {
						labels {
							key
							value
						}
					}
					values {
						id
						value
						traceID
						owner {
							scope
							serviceID
							serviceName
							normal
							serviceInstanceID
							serviceInstanceName
							endpointID
							endpointName
						}
					}
				}
				debuggingTrace {
					traceId
					condition
					duration
					spans {
						spanId
						operation
						msg
						startTime
						endTime
						duration
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"expression": req.Expression,
		"entity":     entity, // Always include entity, even if empty
		"duration": map[string]interface{}{
			"start": duration.Start,
			"end":   duration.End,
			"step":  string(duration.Step),
		},
		// Always provide debug parameters with explicit values
		"debug":     req.Debug,
		"dumpDBRsp": req.DumpDBRsp,
	}

	result, err := executeGraphQLWithContext(ctx, query, variables)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to execute MQE expression: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

// listMQEMetrics lists available metrics
func listMQEMetrics(ctx context.Context, _ *mcp.CallToolRequest, req MQEMetricsListRequest) (*mcp.CallToolResult, any, error) {
	if err := validateMQEMetricsListRequest(&req); err != nil {
		return ResultError(err.Error()), nil, nil
	}

	// GraphQL query for listing metrics
	query := `
		query listMetrics($regex: String) {
			listMetrics(regex: $regex) {
				name
				type
				catalog
			}
		}
	`

	variables := map[string]interface{}{}
	if req.Regex != "" {
		variables["regex"] = req.Regex
	}

	result, err := executeGraphQLWithContext(ctx, query, variables)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to list metrics: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

// getMQEMetricsType gets metric type information
func getMQEMetricsType(ctx context.Context, _ *mcp.CallToolRequest, req MQEMetricsTypeRequest) (*mcp.CallToolResult, any, error) {
	if req.MetricName == "" {
		return ResultError("metric_name must be provided"), nil, nil
	}
	if err := validateMetricName(req.MetricName); err != nil {
		return ResultError(err.Error()), nil, nil
	}

	// GraphQL query for getting metric type
	query := `
		query typeOfMetrics($name: String!) {
			typeOfMetrics(name: $name)
		}
	`

	variables := map[string]interface{}{
		"name": req.MetricName,
	}

	result, err := executeGraphQLWithContext(ctx, query, variables)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to get metric type: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

func validateMQEExpressionRequest(req *MQEExpressionRequest) error {
	if err := validateMQEExpression(req.Expression); err != nil {
		return err
	}

	for fieldName, value := range map[string]string{
		"service_name":               req.ServiceName,
		"service_instance_name":      req.ServiceInstanceName,
		"endpoint_name":              req.EndpointName,
		"process_name":               req.ProcessName,
		"dest_service_name":          req.DestServiceName,
		"dest_service_instance_name": req.DestServiceInstanceName,
		"dest_endpoint_name":         req.DestEndpointName,
		"dest_process_name":          req.DestProcessName,
	} {
		if err := validateMQETextField(fieldName, value, maxMQEEntityFieldLen); err != nil {
			return err
		}
	}

	if err := validateLayerField("layer", req.Layer); err != nil {
		return err
	}
	if err := validateLayerField("dest_layer", req.DestLayer); err != nil {
		return err
	}

	return nil
}

func validateMQEMetricsListRequest(req *MQEMetricsListRequest) error {
	if req == nil || req.Regex == "" {
		return nil
	}
	if err := validateMQETextField("regex", req.Regex, maxMQERegexLength); err != nil {
		return err
	}
	if err := validateRegexComplexity(req.Regex); err != nil {
		return err
	}
	return nil
}

const maxRegexNodes = 50

// validateRegexComplexity rejects patterns with excessive AST node counts.
func validateRegexComplexity(pattern string) error {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return fmt.Errorf("regex is invalid: %w", err)
	}
	if regexNodeCount(re) > maxRegexNodes {
		return fmt.Errorf("regex is too complex")
	}
	return nil
}

func regexNodeCount(re *syntax.Regexp) int {
	count := 1
	for _, sub := range re.Sub {
		count += regexNodeCount(sub)
	}
	return count
}

func validateLayerField(fieldName, value string) error {
	if value == "" {
		return nil
	}
	if err := validateMQETextField(fieldName, value, maxMQEEntityFieldLen); err != nil {
		return err
	}
	if !layerPattern.MatchString(value) {
		return fmt.Errorf("%s contains invalid characters: only uppercase letters, digits, and underscores are allowed", fieldName)
	}
	return nil
}

func validateMetricName(metricName string) error {
	if err := validateMQETextField("metric_name", metricName, maxMetricNameLength); err != nil {
		return err
	}
	if !metricNamePattern.MatchString(metricName) {
		return fmt.Errorf("metric_name contains invalid characters")
	}
	return nil
}

func validateMQEExpression(expression string) error {
	if !utf8.ValidString(expression) {
		return fmt.Errorf("expression must be valid UTF-8")
	}
	if len(expression) > maxMQEExpressionLength {
		return fmt.Errorf("expression exceeds maximum length of %d characters", maxMQEExpressionLength)
	}
	if containsUnsafeControlChars(expression) {
		return fmt.Errorf("expression contains invalid control characters")
	}
	if nestingDepth(expression) > maxMQEExpressionDepth {
		return fmt.Errorf("expression exceeds maximum nesting depth of %d", maxMQEExpressionDepth)
	}
	return nil
}

func validateMQETextField(fieldName, value string, maxLen int) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", fieldName)
	}
	if len(value) > maxLen {
		return fmt.Errorf("%s exceeds maximum length of %d characters", fieldName, maxLen)
	}
	if containsUnsafeControlChars(value) {
		return fmt.Errorf("%s contains invalid control characters", fieldName)
	}
	return nil
}

func containsUnsafeControlChars(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return true
		}
	}
	return false
}

func nestingDepth(value string) int {
	depth := 0
	maxDepth := 0
	for _, r := range value {
		switch r {
		case '(', '{', '[':
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case ')', '}', ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return maxDepth
}

func mqeExpressionTool() *mcp.Tool {
	schema := InferSchema[MQEExpressionRequest]()
	WithRequired(schema, "expression")
	WithEnum(schema, "step", "SECOND", "MINUTE", "HOUR", "DAY", "MONTH")
	WithDescriptions(schema, map[string]string{
		"expression": "MQE expression to execute (required). " +
			"Examples: `service_sla`, `avg(service_cpm)`, `service_sla * 100`, `service_percentile{p='50,75,90,95,99'}`",
		"service_name": "Service name for entity filtering",
		"layer": "Service layer for entity filtering. " +
			"Examples: `GENERAL` (default), `MESH`, `K8S_SERVICE`, `DATABASE`, `VIRTUAL_DATABASE`. " +
			"Defaults to GENERAL if not specified",
		"service_instance_name": "Service instance name for entity filtering",
		"endpoint_name":         "Endpoint name for entity filtering",
		"process_name":          "Process name for entity filtering",
		"normal": "Whether the service is normal (has agent installed). " +
			"If not specified, will be auto-detected based on service layer",
		"dest_service_name": "Destination service name for relation metrics",
		"dest_layer": "Destination service layer for relation metrics. " +
			"Examples: `GENERAL`, `MESH`, `K8S_SERVICE`, `DATABASE`",
		"dest_service_instance_name": "Destination service instance name for relation metrics",
		"dest_endpoint_name":         "Destination endpoint name for relation metrics",
		"dest_process_name":          "Destination process name for relation metrics",
		"dest_normal":                "Whether the destination service is normal",
		"start":                      "Start time for the query. Examples: `2025-07-06 12:00:00`, `-1h` (1 hour ago), `-30m` (30 minutes ago)",
		"end":                        "End time for the query. Examples: `2025-07-06 13:00:00`, `now`, `-10m` (10 minutes ago)",
		"step": "Time step between start time and end time: " +
			"SECOND (second-level), MINUTE (minute-level), HOUR (hour-level), " +
			"DAY (day-level), MONTH (month-level). " +
			"If not specified, uses adaptive step sizing: " +
			"SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)",
		"cold":        "Whether to query from cold-stage storage",
		"debug":       "Enable query tracing and debugging",
		"dump_db_rsp": "Dump database response for debugging",
	})

	return &mcp.Tool{
		Name: "execute_mqe_expression",
		Description: `Execute MQE (Metrics Query Expression) to query and calculate metrics data.

MQE is SkyWalking's powerful query language that allows you to:
- Query metrics with labels: service_percentile{p='50,75,90,95,99'}
- Perform calculations: service_sla * 100, service_cpm / 60
- Compare values: service_resp_time > 3000
- Use aggregations: avg(service_cpm), sum(service_cpm), max(service_resp_time)
- Mathematical functions: round(service_cpm / 60, 2), abs(service_resp_time - 1000)
- TopN queries: top_n(service_cpm, 10, des)
- Trend analysis: increase(service_cpm, 2), rate(service_cpm, 5)
- Sort operations: sort_values(service_resp_time, 10, des)
- Baseline comparison: baseline(service_resp_time, upper)
- Relabel operations: relabels(service_percentile{p='50,75,90,95,99'}, p='50,75,90,95,99', percentile='P50,P75,P90,P95,P99')
- Logical operations: view_as_seq([metric1, metric2]), is_present([metric1, metric2])
- Label aggregation: aggregate_labels(total_commands_rate, sum)

Result Types:
- SINGLE_VALUE: Single metric value (e.g., avg(), sum())
- TIME_SERIES_VALUES: Time series data with timestamps
- SORTED_LIST: Sorted metric values (e.g., top_n())
- RECORD_LIST: Record-based metrics
- LABELED_VALUE: Metrics with multiple labels

USAGE REQUIREMENTS:
- The 'expression' parameter is mandatory for all queries
- For service-specific queries, specify 'service_name' and optionally 'layer' (defaults to GENERAL)
- For relation metrics, provide both source and destination entity parameters
- Use 'start' and 'end' to set a time range; if omitted, defaults to the last 30 minutes
- Use 'debug: true' for query tracing and troubleshooting
- Use 'cold: true' to query from cold storage (BanyanDB only)

Entity Filtering (all optional):
- Service level: service_name + layer + normal
- Instance level: service_instance_name
- Endpoint level: endpoint_name
- Process level: process_name
- Relation queries: dest_service_name + dest_layer, dest_service_instance_name, etc.

Examples:
- {expression: "service_sla * 100", service_name: "Your_ApplicationName", layer: "GENERAL",
  start: "-1h", end: "now"}: Convert SLA to percentage for last hour
- {expression: "service_resp_time > 3000 && service_cpm < 1000", service_name: "Your_ApplicationName", 
	start: "-30m", end: "now"}: Find high latency with low traffic in last 30 minutes
- {expression: "avg(service_cpm)", start: "-2h", end: "now"}: Calculate average CPM for last 2 hours
- {expression: "service_cpm", start: "now", end: "+24h"}: Query CPM for next 24 hours (useful for capacity planning)
- {expression: "top_n(service_cpm, 10, des)", start: "2025-07-06 16:00:00", end: "2025-07-06 17:00:00", 
	step: "MINUTE"}: Top 10 services by CPM with minute granularity`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "execute_mqe_expression", IdempotentHint: true},
	}
}

func mqeMetricsListTool() *mcp.Tool {
	schema := InferSchema[MQEMetricsListRequest]()
	WithDescriptions(schema, map[string]string{
		"regex": "Optional regex pattern to filter metrics by name. Examples: `service_.*`, `.*_cpm`, `endpoint_.*`",
	})

	return &mcp.Tool{
		Name: "list_mqe_metrics",
		Description: `List available metrics in SkyWalking that can be used in MQE expressions.

This tool helps you discover what metrics are available for querying and their metadata information 
including metric type and catalog. You can optionally provide a regex pattern to filter the metrics by name.

Metric Categories:
- Service metrics: service_sla, service_cpm, service_resp_time, service_apdex, service_percentile
- Instance metrics: service_instance_sla, service_instance_cpm, service_instance_resp_time
- Endpoint metrics: endpoint_sla, endpoint_cpm, endpoint_resp_time, endpoint_percentile
- Relation metrics: service_relation_client_cpm, service_relation_server_cpm
- Database metrics: database_access_resp_time, database_access_cpm
- Infrastructure metrics: service_cpu, service_memory, service_thread_count

Metric Types:
- REGULAR_VALUE: Single value metrics (e.g., service_sla, service_cpm)
- LABELED_VALUE: Multi-label metrics (e.g., service_percentile, k8s_cluster_deployment_status)
- SAMPLED_RECORD: Record-based metrics

Usage Tips:
- Use regex patterns to filter specific metric categories
- Check metric type to understand how to use them in MQE expressions
- Regular value metrics can be used directly in calculations
- Labeled value metrics require label selectors: metric_name{label='value'}

Examples:
- {regex: "service_.*"}: List all service-related metrics
- {regex: ".*_cpm"}: List all CPM (calls per minute) metrics
- {regex: ".*percentile.*"}: List all percentile metrics
- {}: List all available metrics`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "list_mqe_metrics", IdempotentHint: true},
	}
}

func mqeMetricsTypeTool() *mcp.Tool {
	schema := InferSchema[MQEMetricsTypeRequest]()
	WithRequired(schema, "metric_name")
	WithDescriptions(schema, map[string]string{
		"metric_name": "Name of the metric to get type information for (required). " +
			"Examples: `service_sla`, `service_percentile`, `endpoint_cpm`",
	})

	return &mcp.Tool{
		Name: "get_mqe_metric_type",
		Description: `Get type information for a specific metric.

This tool returns the type and catalog information for a given metric name, which helps understand 
what kind of data the metric contains and how it should be used in MQE expressions.

Metric Types:
- REGULAR_VALUE: Single numeric value metrics
  - Can be used directly in arithmetic operations
  - Examples: service_sla, service_cpm, service_resp_time
  - Usage: service_sla, service_sla * 100, avg(service_cpm)

- LABELED_VALUE: Multi-dimensional metrics with labels
  - Require label selectors to specify which values to query
  - Examples: service_percentile, k8s_cluster_deployment_status
  - Usage: service_percentile{p='50,75,90,95,99'}

- SAMPLED_RECORD: Record-based metrics with sampling
  - Used for detailed record analysis
  - Examples: top_n_database_statement, traces
  - Usage: Complex aggregations and filtering

Understanding metric types is crucial for:
- Writing correct MQE expressions
- Knowing whether to use label selectors
- Understanding result data structure
- Choosing appropriate aggregation functions

Examples:
- {metric_name: "service_cpm"}: Get type info for service CPM metric
- {metric_name: "service_percentile"}: Get type info for service percentile metric
- {metric_name: "endpoint_sla"}: Get type info for endpoint SLA metric`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "get_mqe_metric_type", IdempotentHint: true},
	}
}
