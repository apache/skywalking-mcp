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
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	api "skywalking.apache.org/repo/goapi/query"

	"github.com/apache/skywalking-cli/pkg/graphql/metrics"
)

// AddMetricsTools registers metrics-related tools with the MCP server
func AddMetricsTools(mcp *server.MCPServer) {
	SingleMetricsTool.Register(mcp)
	TopNMetricsTool.Register(mcp)
}

// Error messages
const (
	ErrMissingMetricsName   = "missing required parameter: metrics_name"
	ErrInvalidTopN          = "top_n must be a positive integer"
	ErrFailedToQueryMetrics = "failed to query metrics: %v"
)

// SingleMetricsRequest defines the parameters for the single metrics tool
type SingleMetricsRequest struct {
	MetricsName             string `json:"metrics_name"`
	Scope                   string `json:"scope,omitempty"`
	ServiceName             string `json:"service_name,omitempty"`
	ServiceInstanceName     string `json:"service_instance_name,omitempty"`
	EndpointName            string `json:"endpoint_name,omitempty"`
	ProcessName             string `json:"process_name,omitempty"`
	DestServiceName         string `json:"dest_service_name,omitempty"`
	DestServiceInstanceName string `json:"dest_service_instance_name,omitempty"`
	DestEndpointName        string `json:"dest_endpoint_name,omitempty"`
	DestProcessName         string `json:"dest_process_name,omitempty"`
	Duration                string `json:"duration,omitempty"`
	Start                   string `json:"start,omitempty"`
	End                     string `json:"end,omitempty"`
	Step                    string `json:"step,omitempty"`
	Cold                    bool   `json:"cold,omitempty"`
}

// TopNMetricsRequest defines the parameters for the top N metrics tool
// ParentService and Normal are used for service/entity identification, matching swctl behavior.
type TopNMetricsRequest struct {
	MetricsName   string `json:"metrics_name"`
	TopN          int    `json:"top_n"`
	Order         string `json:"order,omitempty"`
	Scope         string `json:"scope,omitempty"`
	ServiceID     string `json:"service_id,omitempty"`
	ServiceName   string `json:"service_name,omitempty"`
	ParentService string `json:"parent_service,omitempty"`
	Normal        bool   `json:"normal,omitempty"`
	Duration      string `json:"duration,omitempty"`
	Start         string `json:"start,omitempty"`
	End           string `json:"end,omitempty"`
	Step          string `json:"step,omitempty"`
	Cold          bool   `json:"cold,omitempty"`
}

// MetricsValue represents the result of metrics query
type MetricsValue struct {
	Value int `json:"value"`
}

// ParseScopeInTop infers the scope for topN metrics based on metricsName
func ParseScopeInTop(metricsName string) api.Scope {
	scope := api.ScopeService
	if strings.HasPrefix(metricsName, "service_instance") {
		scope = api.ScopeServiceInstance
	} else if strings.HasPrefix(metricsName, "endpoint") {
		scope = api.ScopeEndpoint
	}
	return scope
}

// validateSingleMetricsRequest validates single metrics request parameters
func validateSingleMetricsRequest(req *SingleMetricsRequest) error {
	if req.MetricsName == "" {
		return errors.New(ErrMissingMetricsName)
	}
	return nil
}

// validateTopNMetricsRequest validates top N metrics request parameters
func validateTopNMetricsRequest(req *TopNMetricsRequest) error {
	if req.MetricsName == "" {
		return errors.New(ErrMissingMetricsName)
	}
	// Set default top_n to 5 if not provided
	if req.TopN == 0 {
		req.TopN = 5
	}
	if req.TopN <= 0 {
		return errors.New(ErrInvalidTopN)
	}
	return nil
}

// buildTopNCondition builds the top N condition from request parameters
func buildTopNCondition(req *TopNMetricsRequest) *api.TopNCondition {
	parentService := ""
	normal := false
	// Parse service-id if present, otherwise use ServiceName if provided
	if req.ServiceID != "" {
		var err error
		parentService, normal, err = ParseServiceID(req.ServiceID)
		if err != nil {
			parentService = ""
			normal = false
		}
	} else if req.ServiceName != "" {
		parentService = req.ServiceName
	}

	condition := &api.TopNCondition{
		Name:          req.MetricsName,
		ParentService: &parentService,
		Normal:        &normal,
		TopN:          req.TopN,
		Order:         api.OrderDes,
	}
	if req.Order != "" {
		order := api.Order(req.Order)
		if order.IsValid() {
			condition.Order = order
		}
	}
	// Always set scope, using ParseScopeInTop if not provided
	var scope api.Scope
	if req.Scope != "" {
		scope = api.Scope(req.Scope)
	} else {
		scope = ParseScopeInTop(req.MetricsName)
	}
	condition.Scope = &scope

	return condition
}

// ParseServiceID decodes a service id into service name and normal flag
func ParseServiceID(id string) (name string, isNormal bool, err error) {
	if id == "" {
		return "", false, nil
	}
	parts := strings.Split(id, ".")
	if len(parts) != 2 {
		return "", false, fmt.Errorf("invalid service id, cannot be splitted into 2 parts. %v", id)
	}
	nameBytes, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false, err
	}
	name = string(nameBytes)
	isNormal = parts[1] == "1"
	return name, isNormal, nil
}

// buildMetricsCondition builds the metrics condition from request parameters
func buildMetricsCondition(req *SingleMetricsRequest) *api.MetricsCondition {
	condition := &api.MetricsCondition{
		Name: req.MetricsName,
	}

	entity := &api.Entity{}
	if req.Scope != "" {
		scope := api.Scope(req.Scope)
		entity.Scope = &scope
	}
	if req.ServiceName != "" {
		entity.ServiceName = &req.ServiceName
	}
	if req.ServiceInstanceName != "" {
		entity.ServiceInstanceName = &req.ServiceInstanceName
	}
	if req.EndpointName != "" {
		entity.EndpointName = &req.EndpointName
	}
	if req.ProcessName != "" {
		entity.ProcessName = &req.ProcessName
	}
	if req.DestServiceName != "" {
		entity.DestServiceName = &req.DestServiceName
	}
	if req.DestServiceInstanceName != "" {
		entity.DestServiceInstanceName = &req.DestServiceInstanceName
	}
	if req.DestEndpointName != "" {
		entity.DestEndpointName = &req.DestEndpointName
	}
	if req.DestProcessName != "" {
		entity.DestProcessName = &req.DestProcessName
	}
	condition.Entity = entity
	return condition
}

// querySingleMetrics queries single-value metrics
func querySingleMetrics(ctx context.Context, req *SingleMetricsRequest) (*mcp.CallToolResult, error) {
	if err := validateSingleMetricsRequest(req); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	condition := buildMetricsCondition(req)

	var duration api.Duration
	if req.Duration != "" {
		duration = ParseDuration(req.Duration, req.Cold)
	} else {
		duration = BuildDuration(req.Start, req.End, req.Step, req.Cold, 0)
	}

	value, err := metrics.IntValues(ctx, *condition, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrFailedToQueryMetrics, err)), nil
	}
	result := MetricsValue{Value: value}
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// queryTopNMetrics queries top N metrics
func queryTopNMetrics(ctx context.Context, req *TopNMetricsRequest) (*mcp.CallToolResult, error) {
	if err := validateTopNMetricsRequest(req); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	condition := buildTopNCondition(req)

	// Set default duration if none provided
	if req.Duration == "" && req.Start == "" && req.End == "" {
		req.Duration = "30m"
	}

	var duration api.Duration
	if req.Duration != "" {
		duration = ParseDuration(req.Duration, req.Cold)
	} else {
		duration = BuildDuration(req.Start, req.End, req.Step, req.Cold, 0)
	}

	values, err := metrics.SortMetrics(ctx, *condition, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrFailedToQueryMetrics, err)), nil
	}
	jsonBytes, err := json.Marshal(values)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// SingleMetricsTool is a tool for querying single-value metrics
var SingleMetricsTool = NewTool[SingleMetricsRequest, *mcp.CallToolResult](
	"query_single_metrics",
	`This tool queries single-value metrics defined in backend OAL from SkyWalking OAP.

Workflow:
1. Use this tool when you need to get a single metric value for a specific entity
2. Specify the metrics name and entity details (service, endpoint, etc.)
3. Set the time range for the query
4. Get the metric value as a single integer result

Metrics Examples:
- service_cpm: Calls per minute for a service
- endpoint_cpm: Calls per minute for an endpoint
- service_resp_time: Response time for a service
- service_apdex: Apdex score for a service
- service_sla: SLA percentage for a service

Entity Scopes:
- Service: Service-level metrics
- ServiceInstance: Service instance-level metrics
- Endpoint: Endpoint-level metrics
- Process: Process-level metrics
- ServiceRelation: Service relationship metrics
- ServiceInstanceRelation: Service instance relationship metrics
- EndpointRelation: Endpoint relationship metrics
- ProcessRelation: Process relationship metrics

Time Format:
- Absolute time: "2023-01-01 12:00:00", "2023-01-01 12"
- Relative time: "-30m" (30 minutes ago), "-1h" (1 hour ago)
- Step: "SECOND", "MINUTE", "HOUR", "DAY"

Examples:
- {"metrics_name": "service_cpm", "service_name": "business-zone::projectC", "duration": "-1h"}: Get calls per minute for a service in the past hour
- {"metrics_name": "endpoint_cpm", "service_name": "business-zone::projectC", 
  "endpoint_name": "/projectC/{value}", "duration": "-30m"}: Get calls per minute for a specific endpoint in the past 30 minutes
- {"metrics_name": "service_resp_time", "service_name": "web-service", 
  "start": "-1h", "end": "now", "step": "MINUTE"}: Get service response time with custom time range
- {"metrics_name": "service_apdex", "service_name": "api-gateway", "cold": true}: Get Apdex score from cold storage`,
	querySingleMetrics,
	mcp.WithTitleAnnotation("Query single-value metrics"),
	mcp.WithString("metrics_name", mcp.Required(),
		mcp.Description(`The name of the metrics to query. Examples: service_sla, endpoint_sla, 
service_instance_sla, service_cpm, service_resp_time, service_apdex`),
	),
	mcp.WithString("scope",
		mcp.Enum(string(api.ScopeAll), string(api.ScopeService), string(api.ScopeServiceInstance), string(api.ScopeEndpoint), string(api.ScopeProcess),
			string(api.ScopeServiceRelation), string(api.ScopeServiceInstanceRelation), string(api.ScopeEndpointRelation), string(api.ScopeProcessRelation)),
		mcp.Description(`The scope of the metrics entity:
- 'Service': Service-level metrics (default)
- 'ServiceInstance': Service instance-level metrics
- 'Endpoint': Endpoint-level metrics
- 'Process': Process-level metrics
- 'ServiceRelation': Service relationship metrics
- 'ServiceInstanceRelation': Service instance relationship metrics
- 'EndpointRelation': Endpoint relationship metrics
- 'ProcessRelation': Process relationship metrics`),
	),
	mcp.WithString("service_name",
		mcp.Description("Service name to filter metrics. Use this to get metrics for a specific service."),
	),
	mcp.WithString("service_instance_name",
		mcp.Description("Service instance name to filter metrics. Use this to get metrics for a specific service instance."),
	),
	mcp.WithString("endpoint_name",
		mcp.Description("Endpoint name to filter metrics. Use this to get metrics for a specific endpoint."),
	),
	mcp.WithString("process_name",
		mcp.Description("Process name to filter metrics. Use this to get metrics for a specific process."),
	),
	mcp.WithString("dest_service_name",
		mcp.Description("Destination service name for relationship metrics. Use this for service relation scopes."),
	),
	mcp.WithString("dest_service_instance_name",
		mcp.Description("Destination service instance name for relationship metrics. Use this for service instance relation scopes."),
	),
	mcp.WithString("dest_endpoint_name",
		mcp.Description("Destination endpoint name for relationship metrics. Use this for endpoint relation scopes."),
	),
	mcp.WithString("dest_process_name",
		mcp.Description("Destination process name for relationship metrics. Use this for process relation scopes."),
	),
	mcp.WithString("duration",
		mcp.Description("Time duration for the query relative to current time. "+
			"Negative values query the past: \"-1h\" (past 1 hour), \"-30m\" (past 30 minutes), \"-7d\" (past 7 days). "+
			"Positive values query the future: \"1h\" (next 1 hour), \"24h\" (next 24 hours)"),
	),
	mcp.WithString("start",
		mcp.Description("Start time for the query. Examples: \"2023-01-01 12:00:00\", \"-1h\" (1 hour ago), \"-30m\" (30 minutes ago)"),
	),
	mcp.WithString("end",
		mcp.Description("End time for the query. Examples: \"2023-01-01 13:00:00\", \"now\", \"-10m\" (10 minutes ago)"),
	),
	mcp.WithString("step",
		mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description(`Time step between start time and end time:
|- 'SECOND': Second-level granularity
|- 'MINUTE': Minute-level granularity
|- 'HOUR': Hour-level granularity
|- 'DAY': Day-level granularity
If not specified, uses adaptive step sizing: 
SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)`),
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)

// TopNMetricsTool is a tool for querying top N metrics
var TopNMetricsTool = NewTool[TopNMetricsRequest, *mcp.CallToolResult](
	"query_top_n_metrics",
	`This tool queries the top N entities sorted by the specified metrics from SkyWalking OAP.

Workflow:
1. Use this tool when you need to find the top N entities based on a specific metric
2. Specify the metrics name and the number of top entities to retrieve
3. Set the time range for the query
4. Get a list of top N entities with their metric values

Metrics Examples:
- service_sla: SLA percentage for services
- endpoint_sla: SLA percentage for endpoints
- service_instance_sla: SLA percentage for service instances
- service_cpm: Calls per minute for services
- service_resp_time: Response time for services
- service_apdex: Apdex score for services

Entity Scopes:
- Service: Service-level metrics (default)
- ServiceInstance: Service instance-level metrics
- Endpoint: Endpoint-level metrics
- Process: Process-level metrics

Order Options:
- ASC: Ascending order (lowest values first)
- DES: Descending order (highest values first, default)

Time Format:
- Absolute time: "2023-01-01 12:00:00", "2023-01-01 12"
- Relative time: "-30m" (30 minutes ago), "-1h" (1 hour ago)
- Step: "SECOND", "MINUTE", "HOUR", "DAY"

Examples:
- {"metrics_name": "service_sla", "top_n": 5, "duration": "-1h"}: Get top 5 services with highest SLA in the past hour
- {"metrics_name": "endpoint_sla", "top_n": 10, "order": "ASC", "duration": "-30m"}: Get top 10 endpoints with lowest SLA in the past 30 minutes
- {"metrics_name": "service_instance_sla", "top_n": 3, "service_name": "boutique::adservice", 
  "duration": "-1h"}: Get top 3 instances of a specific service with highest SLA in the past hour
- {"metrics_name": "service_cpm", "top_n": 5, "start": "-1h", "end": "now", 
  "step": "MINUTE"}: Get top 5 services with highest calls per minute with custom time range`,
	queryTopNMetrics,
	mcp.WithTitleAnnotation("Query top N metrics"),
	mcp.WithString("metrics_name", mcp.Required(),
		mcp.Description(`The name of the metrics to query. Examples: service_sla, endpoint_sla, 
service_instance_sla, service_cpm, service_resp_time, service_apdex`),
	),
	mcp.WithNumber("top_n", mcp.Required(),
		mcp.Description("The number of top entities to retrieve. Must be a positive integer."),
	),
	mcp.WithString("order",
		mcp.Enum("ASC", "DES"),
		mcp.Description(`The order by which the top entities are sorted:
- 'ASC': Ascending order (lowest values first)
- 'DES': Descending order (highest values first, default)`),
	),
	mcp.WithString("scope",
		mcp.Enum(string(api.ScopeAll), string(api.ScopeService), string(api.ScopeServiceInstance), string(api.ScopeEndpoint), string(api.ScopeProcess)),
		mcp.Description(`The scope of the metrics entity:
- 'Service': Service-level metrics (default)
- 'ServiceInstance': Service instance-level metrics
- 'Endpoint': Endpoint-level metrics
- 'Process': Process-level metrics`),
	),
	mcp.WithString("service_name",
		mcp.Description("Parent service name to filter metrics. Use this to get top N entities within a specific service."),
	),
	mcp.WithString("duration",
		mcp.Description("Time duration for the query relative to current time. "+
			"Negative values query the past: \"-1h\" (past 1 hour), \"-30m\" (past 30 minutes), \"-7d\" (past 7 days). "+
			"Positive values query the future: \"1h\" (next 1 hour), \"24h\" (next 24 hours)"),
	),
	mcp.WithString("start",
		mcp.Description("Start time for the query. Examples: \"2023-01-01 12:00:00\", \"-1h\" (1 hour ago), \"-30m\" (30 minutes ago)"),
	),
	mcp.WithString("end",
		mcp.Description("End time for the query. Examples: \"2023-01-01 13:00:00\", \"now\", \"-10m\" (10 minutes ago)"),
	),
	mcp.WithString("step",
		mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description(`Time step between start time and end time:
|- 'SECOND': Second-level granularity
|- 'MINUTE': Minute-level granularity
|- 'HOUR': Hour-level granularity
|- 'DAY': Day-level granularity
If not specified, uses adaptive step sizing: 
SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)`),
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)
