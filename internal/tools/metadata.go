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
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/apache/skywalking-cli/pkg/graphql/metadata"
	api "skywalking.apache.org/repo/goapi/query"
)

// AddMetadataTools registers metadata-related tools with the MCP server
func AddMetadataTools(s *server.MCPServer) {
	ListLayersTool.Register(s)
	ListServicesTool.Register(s)
	ListInstancesTool.Register(s)
	ListEndpointsTool.Register(s)
	ListProcessesTool.Register(s)
}

// ListLayersRequest defines the parameters for the list_layers tool (no parameters needed)
type ListLayersRequest struct{}

// listLayers queries available layers from SkyWalking OAP
func listLayers(ctx context.Context, _ *ListLayersRequest) (*mcp.CallToolResult, error) {
	layers, err := metadata.ListLayers(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list layers: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(layers)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListEndpointsRequest defines the parameters for the list_endpoints tool
type ListEndpointsRequest struct {
	ServiceID string `json:"service_id"`
	Keyword   string `json:"keyword"`
	Limit     int    `json:"limit"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Step      string `json:"step"`
	Cold      bool   `json:"cold"`
}

// listEndpoints searches endpoints for a given service from SkyWalking OAP
func listEndpoints(ctx context.Context, req *ListEndpointsRequest) (*mcp.CallToolResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	timeCtx := GetTimeContext(ctx)
	var durationPtr *api.Duration
	if req.Start != "" || req.End != "" {
		d := BuildDurationWithContext(req.Start, req.End, req.Step, req.Cold, DefaultDuration, timeCtx)
		durationPtr = &d
	}
	endpoints, err := metadata.SearchEndpoints(ctx, req.ServiceID, req.Keyword, limit, durationPtr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list endpoints: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(endpoints)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListEndpointsTool lists endpoints of a service in SkyWalking OAP
var ListEndpointsTool = NewTool(
	"list_endpoints",
	`List endpoints of a service registered in SkyWalking OAP.

An endpoint represents an individual API path or operation exposed by a service.
Use list_services to obtain a service ID before calling this tool.

The response includes each endpoint's id and name.
The id can be used as a filter in metrics queries that accept an endpoint scope.

Workflow:
1. Call list_layers to find available layers
2. Call list_services with a layer to find the service and its ID
3. Call this tool with the service ID to list its endpoints

Examples:
- {"service_id": "abc123"}: List all endpoints of a service
- {"service_id": "abc123", "keyword": "/api/user", "limit": 20}: Search endpoints by keyword`,
	listEndpoints,
	mcp.WithTitleAnnotation("List service endpoints"),
	mcp.WithString("service_id", mcp.Required(),
		mcp.Description("The service ID to list endpoints for. Use list_services to obtain a service ID."),
	),
	mcp.WithString("keyword",
		mcp.Description("Keyword to filter endpoints by name. Leave empty to list all endpoints."),
	),
	mcp.WithNumber("limit",
		mcp.Description("Maximum number of endpoints to return. Defaults to 100."),
	),
	mcp.WithString("start",
		mcp.Description(`Start time for the query. Examples: "2024-01-01 12:00:00", "-1h" (1 hour ago).`),
	),
	mcp.WithString("end",
		mcp.Description(`End time for the query. Examples: "2024-01-01 13:00:00", "now". Defaults to current time if omitted.`),
	),
	mcp.WithString("step",
		mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive sizing: SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)."),
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)

// ListProcessesRequest defines the parameters for the list_processes tool
type ListProcessesRequest struct {
	InstanceID string `json:"instance_id"`
	Start      string `json:"start"`
	End        string `json:"end"`
	Step       string `json:"step"`
	Cold       bool   `json:"cold"`
}

// listProcesses queries processes for a given service instance from SkyWalking OAP
func listProcesses(ctx context.Context, req *ListProcessesRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, req.Cold, DefaultDuration, timeCtx)
	processes, err := metadata.Processes(ctx, req.InstanceID, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list processes: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(processes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListProcessesTool lists processes of a service instance in SkyWalking OAP
var ListProcessesTool = NewTool(
	"list_processes",
	`List processes of a service instance registered in SkyWalking OAP.

A process represents an individual OS or language-level process running within a service instance.
Use list_instances to obtain an instance ID before calling this tool.

The response includes each process's id, name, serviceId, serviceName, instanceId, instanceName,
agentId, detectType, labels, and attributes.

Workflow:
1. Call list_layers to find available layers
2. Call list_services with a layer to find the service and its ID
3. Call list_instances with the service ID to find an instance and its ID
4. Call this tool with the instance ID and a time range to list its processes

Examples:
- {"instance_id": "abc123", "start": "-1h"}: List processes active in the past hour
- {"instance_id": "abc123", "start": "2024-01-01 12:00", "end": "2024-01-01 13:00"}: List processes in a time range`,
	listProcesses,
	mcp.WithTitleAnnotation("List instance processes"),
	mcp.WithString("instance_id", mcp.Required(),
		mcp.Description("The instance ID to list processes for. Use list_instances to obtain an instance ID."),
	),
	mcp.WithString("start", mcp.Required(),
		mcp.Description(`Start time for the query. Examples: "2024-01-01 12:00:00", "-1h" (1 hour ago), "-30m" (30 minutes ago).`),
	),
	mcp.WithString("end",
		mcp.Description(`End time for the query. Examples: "2024-01-01 13:00:00", "now". Defaults to current time if omitted.`),
	),
	mcp.WithString("step",
		mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive sizing: SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)."),
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)

// ListInstancesRequest defines the parameters for the list_instances tool
type ListInstancesRequest struct {
	ServiceID string `json:"service_id"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Step      string `json:"step"`
	Cold      bool   `json:"cold"`
}

// listInstances queries service instances from SkyWalking OAP
func listInstances(ctx context.Context, req *ListInstancesRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, req.Cold, DefaultDuration, timeCtx)
	instances, err := metadata.Instances(ctx, req.ServiceID, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list instances: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(instances)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListInstancesTool lists all instances of a service in SkyWalking OAP
var ListInstancesTool = NewTool(
	"list_instances",
	`List all instances of a service registered in SkyWalking OAP.

A service instance represents an individual running process of a service (e.g. a pod or JVM process).
Use list_services to obtain a service ID before calling this tool.

The response includes each instance's id, name, language, instanceUUID, and attributes.
The id can be used as a filter in metrics or log queries that accept a service_instance_id.

Workflow:
1. Call list_layers to find available layers
2. Call list_services with a layer to find the service and its ID
3. Call this tool with the service ID and a time range to list its instances

Examples:
- {"service_id": "abc123", "start": "2024-01-01 12:00", "end": "2024-01-01 13:00"}: List instances in a time range
- {"service_id": "abc123", "start": "-1h"}: List instances active in the past hour`,
	listInstances,
	mcp.WithTitleAnnotation("List service instances"),
	mcp.WithString("service_id", mcp.Required(),
		mcp.Description("The service ID to list instances for. Use list_services to obtain a service ID."),
	),
	mcp.WithString("start", mcp.Required(),
		mcp.Description(`Start time for the query. Examples: "2024-01-01 12:00:00", "-1h" (1 hour ago), "-30m" (30 minutes ago).`),
	),
	mcp.WithString("end",
		mcp.Description(`End time for the query. Examples: "2024-01-01 13:00:00", "now". Defaults to current time if omitted.`),
	),
	mcp.WithString("step",
		mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive sizing: SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)."),
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)

// listServices queries services for a given layer from SkyWalking OAP
func listServices(ctx context.Context, req *ListServicesRequest) (*mcp.CallToolResult, error) {
	services, err := metadata.ListLayerService(ctx, req.Layer)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list services: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(services)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// ListServicesTool lists all services for a given layer in SkyWalking OAP
var ListServicesTool = NewTool(
	"list_services",
	`List all services registered in SkyWalking OAP under a specific layer.

A service represents a logical grouping of monitored workloads. Each service belongs to one
or more layers (e.g. GENERAL, MESH, K8S). Use list_layers first to discover available layers.

The response includes each service's id, name, group, shortName, layers, and normal flag.
The id can be used as a filter in other tools such as query_logs or query_traces.

Workflow:
1. Call list_layers to discover available layers
2. Call this tool with the desired layer to get the services in that layer

Examples:
- {"layer": "GENERAL"}: List all services in the GENERAL layer
- {"layer": "MESH"}: List all services in the service mesh layer`,
	listServices,
	mcp.WithTitleAnnotation("List services by layer"),
	mcp.WithString("layer", mcp.Required(),
		mcp.Description("The layer to list services for. Use list_layers to get available layer names."),
	),
)

// ListLayersTool lists all available layers in SkyWalking OAP
var ListLayersTool = NewTool(
	"list_layers",
	`List all available layers registered in SkyWalking OAP.

A layer represents a technology or deployment environment in SkyWalking's topology,
such as GENERAL, MESH, K8S, OS_LINUX, etc. Layers are used to categorize services
and filter topology views.

Workflow:
1. Call this tool to discover which layers are available in the monitored environment
2. Use the returned layer names when querying services or metrics that require a layer filter

Examples:
- {}: List all layers (no parameters required)`,
	listLayers,
	mcp.WithTitleAnnotation("List available layers"),
)
