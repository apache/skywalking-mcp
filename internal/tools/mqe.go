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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/viper"
	api "skywalking.apache.org/repo/goapi/query"
)

// AddMQETools registers MQE-related tools with the MCP server
func AddMQETools(mcp *server.MCPServer) {
	MQEExpressionTool.Register(mcp)
	MQEMetricsListTool.Register(mcp)
	MQEMetricsTypeTool.Register(mcp)
}

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

// executeGraphQL executes a GraphQL query against SkyWalking OAP
func executeGraphQL(ctx context.Context, url, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	url = FinalizeURL(url)

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP request failed with status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var graphqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		var errorMsgs []string
		for _, err := range graphqlResp.Errors {
			errorMsgs = append(errorMsgs, err.Message)
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(errorMsgs, ", "))
	}

	return &graphqlResp, nil
}

// MQEExpressionRequest represents a request to execute MQE expression
type MQEExpressionRequest struct {
	Expression              string `json:"expression"`
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
	Duration                string `json:"duration,omitempty"`
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
	MetricName string `json:"metric_name"`
}

// ListServicesRequest represents a request to list services
type ListServicesRequest struct {
	Layer string `json:"layer"`
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

	result, err := executeGraphQL(ctx, viper.GetString("url"), query, variables)
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

	result, err := executeGraphQL(ctx, viper.GetString("url"), query, variables)
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
func executeMQEExpression(ctx context.Context, req *MQEExpressionRequest) (*mcp.CallToolResult, error) {
	if req.Expression == "" {
		return mcp.NewToolResultError("expression is required"), nil
	}

	entity := buildMQEEntity(ctx, req)

	var duration api.Duration
	if req.Duration != "" {
		duration = ParseDuration(req.Duration, req.Cold)
	} else {
		duration = BuildDuration(req.Start, req.End, req.Step, req.Cold, DefaultDuration)
	}

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

	result, err := executeGraphQL(ctx, viper.GetString("url"), query, variables)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute MQE expression: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// listMQEMetrics lists available metrics
func listMQEMetrics(ctx context.Context, req *MQEMetricsListRequest) (*mcp.CallToolResult, error) {
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
	if req != nil && req.Regex != "" {
		variables["regex"] = req.Regex
	}

	result, err := executeGraphQL(ctx, viper.GetString("url"), query, variables)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list metrics: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListMQEMetricsInternal is an exported function for internal use by resources package
func ListMQEMetricsInternal(ctx context.Context, regex *string) (string, error) {
	var req *MQEMetricsListRequest
	if regex != nil {
		req = &MQEMetricsListRequest{Regex: *regex}
	}
	result, err := listMQEMetrics(ctx, req)
	if err != nil {
		return "", err
	}

	// Extract the text content from the tool result
	if textResult, ok := result.Content[0].(mcp.TextContent); ok {
		return textResult.Text, nil
	}

	return "", fmt.Errorf("unexpected result format")
}

// getMQEMetricsType gets metric type information
func getMQEMetricsType(ctx context.Context, req *MQEMetricsTypeRequest) (*mcp.CallToolResult, error) {
	if req.MetricName == "" {
		return mcp.NewToolResultError("metric_name must be provided"), nil
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

	result, err := executeGraphQL(ctx, viper.GetString("url"), query, variables)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get metric type: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(result.Data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

var MQEExpressionTool = NewTool[MQEExpressionRequest, *mcp.CallToolResult](
	"execute_mqe_expression",
	`Execute MQE (Metrics Query Expression) to query and calculate metrics data.

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
- Either specify 'duration' OR both 'start' and 'end' for time range
- Use 'debug: true' for query tracing and troubleshooting
- Use 'cold: true' to query from cold storage (BanyanDB only)

Entity Filtering (all optional):
- Service level: service_name + layer + normal
- Instance level: service_instance_name
- Endpoint level: endpoint_name
- Process level: process_name
- Relation queries: dest_service_name + dest_layer, dest_service_instance_name, etc.

Examples:
- {expression: "service_sla * 100", service_name: "Your_ApplicationName", layer: "GENERAL", duration: "-1h"}: Convert SLA to percentage for last hour
- {expression: "service_resp_time > 3000 && service_cpm < 1000", service_name: "Your_ApplicationName", 
  duration: "-30m"}: Find high latency with low traffic in last 30 minutes
- {expression: "avg(service_cpm)", duration: "-2h"}: Calculate average CPM for last 2 hours
- {expression: "service_cpm", duration: "24h"}: Query CPM for next 24 hours (useful for capacity planning)
- {expression: "top_n(service_cpm, 10, des)", start: "2025-07-06 16:00:00", end: "2025-07-06 17:00:00", 
  step: "MINUTE"}: Top 10 services by CPM with minute granularity`,
	executeMQEExpression,
	mcp.WithString("expression", mcp.Required(),
		mcp.Description("MQE expression to execute (required). "+
			"Examples: `service_sla`, `avg(service_cpm)`, `service_sla * 100`, `service_percentile{p='50,75,90,95,99'}`")),
	mcp.WithString("service_name", mcp.Description("Service name for entity filtering")),
	mcp.WithString("layer",
		mcp.Description("Service layer for entity filtering. "+
			"Examples: `GENERAL` (default), `MESH`, `K8S_SERVICE`, `DATABASE`, `VIRTUAL_DATABASE`. "+
			"Defaults to GENERAL if not specified")),
	mcp.WithString("service_instance_name", mcp.Description("Service instance name for entity filtering")),
	mcp.WithString("endpoint_name", mcp.Description("Endpoint name for entity filtering")),
	mcp.WithString("process_name", mcp.Description("Process name for entity filtering")),
	mcp.WithBoolean("normal",
		mcp.Description("Whether the service is normal (has agent installed). "+
			"If not specified, will be auto-detected based on service layer")),
	mcp.WithString("dest_service_name", mcp.Description("Destination service name for relation metrics")),
	mcp.WithString("dest_layer",
		mcp.Description("Destination service layer for relation metrics. "+
			"Examples: `GENERAL`, `MESH`, `K8S_SERVICE`, `DATABASE`")),
	mcp.WithString("dest_service_instance_name", mcp.Description("Destination service instance name for relation metrics")),
	mcp.WithString("dest_endpoint_name", mcp.Description("Destination endpoint name for relation metrics")),
	mcp.WithString("dest_process_name", mcp.Description("Destination process name for relation metrics")),
	mcp.WithBoolean("dest_normal", mcp.Description("Whether the destination service is normal")),
	mcp.WithString("duration",
		mcp.Description("Time duration for the query relative to current time. "+
			"Negative values query the past: `-1h` (past 1 hour), `-30m` (past 30 minutes), `-7d` (past 7 days). "+
			"Positive values query the future: `1h` (next 1 hour), `24h` (next 24 hours). "+
			"Use this OR specify both start+end")),
	mcp.WithString("start", mcp.Description("Start time for the query. Examples: `2025-07-06 12:00:00`, `-1h` (1 hour ago), `-30m` (30 minutes ago)")),
	mcp.WithString("end", mcp.Description("End time for the query. Examples: `2025-07-06 13:00:00`, `now`, `-10m` (10 minutes ago)")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY", "MONTH"),
		mcp.Description("Time step between start time and end time: "+
			"SECOND (second-level), MINUTE (minute-level), HOUR (hour-level), "+
			"DAY (day-level), MONTH (month-level). "+
			"If not specified, uses adaptive step sizing: "+
			"SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)")),
	mcp.WithBoolean("cold", mcp.Description("Whether to query from cold-stage storage")),
	mcp.WithBoolean("debug", mcp.Description("Enable query tracing and debugging")),
	mcp.WithBoolean("dump_db_rsp", mcp.Description("Dump database response for debugging")),
)

var MQEMetricsListTool = NewTool[MQEMetricsListRequest, *mcp.CallToolResult](
	"list_mqe_metrics",
	`List available metrics in SkyWalking that can be used in MQE expressions.

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
	listMQEMetrics,
	mcp.WithString("regex", mcp.Description("Optional regex pattern to filter metrics by name. Examples: `service_.*`, `.*_cpm`, `endpoint_.*`")),
)

var MQEMetricsTypeTool = NewTool[MQEMetricsTypeRequest, *mcp.CallToolResult](
	"get_mqe_metric_type",
	`Get type information for a specific metric.

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
	getMQEMetricsType,
	mcp.WithString("metric_name", mcp.Required(),
		mcp.Description("Name of the metric to get type information for (required). "+
			"Examples: `service_sla`, `service_percentile`, `endpoint_cpm`")),
)
