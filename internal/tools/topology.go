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

	"github.com/machinebox/graphql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	api "skywalking.apache.org/repo/goapi/query"

	"github.com/apache/skywalking-cli/pkg/graphql/client"
	"github.com/apache/skywalking-cli/pkg/graphql/dependency"
)

// AddTopologyTools registers topology-related tools with the MCP server
func AddTopologyTools(s *server.MCPServer) {
	ServicesTopologyTool.Register(s)
	InstancesTopologyTool.Register(s)
	EndpointsTopologyTool.Register(s)
	ProcessesTopologyTool.Register(s)
}

const getServicesTopologyGQL = `
query ($serviceIds: [ID!]!, $duration: Duration!) {
	result: getServicesTopology(serviceIds: $serviceIds, duration: $duration) {
		nodes { id name type isReal layers }
		calls { id source detectPoints target sourceComponents targetComponents }
	}
}`

func servicesTopology(ctx context.Context, serviceIDs []string, duration api.Duration) (api.Topology, error) {
	var response map[string]api.Topology
	request := graphql.NewRequest(getServicesTopologyGQL)
	request.Var("serviceIds", serviceIDs)
	request.Var("duration", duration)
	err := client.ExecuteQuery(ctx, request, &response)
	return response["result"], err
}

type ServicesTopologyRequest struct {
	ServiceIDs []string `json:"service_ids,omitempty"`
	Layer      string   `json:"layer,omitempty"`
	Start      string   `json:"start,omitempty"`
	End        string   `json:"end,omitempty"`
	Step       string   `json:"step,omitempty"`
}

func queryServicesTopology(ctx context.Context, req *ServicesTopologyRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	var (
		topology api.Topology
		err      error
	)

	if len(req.ServiceIDs) > 0 {
		topology, err = servicesTopology(ctx, req.ServiceIDs, duration)
	} else if req.Layer != "" {
		topology, err = dependency.GlobalTopology(ctx, req.Layer, duration)
	} else {
		topology, err = dependency.GlobalTopologyWithoutLayer(ctx, duration)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query topology: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

type InstancesTopologyRequest struct {
	ClientServiceID string `json:"client_service_id"`
	ServerServiceID string `json:"server_service_id"`
	Start           string `json:"start,omitempty"`
	End             string `json:"end,omitempty"`
	Step            string `json:"step,omitempty"`
}

func queryInstancesTopology(ctx context.Context, req *InstancesTopologyRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	topology, err := dependency.InstanceTopology(ctx, req.ClientServiceID, req.ServerServiceID, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query instances topology: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

type EndpointsTopologyRequest struct {
	EndpointID string `json:"endpoint_id"`
	Start      string `json:"start,omitempty"`
	End        string `json:"end,omitempty"`
	Step       string `json:"step,omitempty"`
}

func queryEndpointsTopology(ctx context.Context, req *EndpointsTopologyRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	topology, err := dependency.EndpointDependency(ctx, req.EndpointID, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query endpoints topology: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

type ProcessesTopologyRequest struct {
	ServiceInstanceID string `json:"service_instance_id"`
	Start             string `json:"start,omitempty"`
	End               string `json:"end,omitempty"`
	Step              string `json:"step,omitempty"`
}

func queryProcessesTopology(ctx context.Context, req *ProcessesTopologyRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	topology, err := dependency.ProcessTopology(ctx, req.ServiceInstanceID, duration)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query processes topology: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

var ProcessesTopologyTool = NewTool(
	"query_processes_topology",
	`Query the process topology for a given service instance from SkyWalking OAP (getProcessTopology).

Returns the topology of processes running within the given service instance, including process nodes and the calls between them.

Examples:
- {"service_instance_id": "instance-id-1"}: Process topology for the last 30 minutes
- {"service_instance_id": "instance-id-1", "start": "-1h"}: Process topology for the last hour`,
	queryProcessesTopology,
	mcp.WithString("service_instance_id",
		mcp.Required(),
		mcp.Description("The ID of the service instance to query process topology for.")),
	mcp.WithString("start", mcp.Description("Start time for the query.")),
	mcp.WithString("end", mcp.Description("End time for the query. Default is now.")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive step sizing.")),
)

var EndpointsTopologyTool = NewTool(
	"query_endpoints_topology",
	`Query the endpoint dependency topology for a given endpoint from SkyWalking OAP (getEndpointDependencies).

Returns the topology of endpoints that the given endpoint depends on or is depended upon by, including endpoint nodes and the calls between them.

Examples:
- {"endpoint_id": "ep-id-1"}: Endpoint topology for the last 30 minutes
- {"endpoint_id": "ep-id-1", "start": "-1h"}: Endpoint topology for the last hour`,
	queryEndpointsTopology,
	mcp.WithString("endpoint_id",
		mcp.Required(),
		mcp.Description("The ID of the endpoint to query dependencies for.")),
	mcp.WithString("start", mcp.Description("Start time for the query.")),
	mcp.WithString("end", mcp.Description("End time for the query. Default is now.")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive step sizing.")),
)

var InstancesTopologyTool = NewTool(
	"query_instances_topology",
	`Query the service instance topology between two services from SkyWalking OAP (getServiceInstanceTopology).

Returns the topology of service instances for the given client and server services, including instance nodes and the calls between them.

Examples:
- {"client_service_id": "svc-id-1", "server_service_id": "svc-id-2"}: Instance topology for the last 30 minutes
- {"client_service_id": "svc-id-1", "server_service_id": "svc-id-2", "start": "-1h"}: Instance topology for the last hour`,
	queryInstancesTopology,
	mcp.WithString("client_service_id",
		mcp.Required(),
		mcp.Description("The ID of the client (upstream) service.")),
	mcp.WithString("server_service_id",
		mcp.Required(),
		mcp.Description("The ID of the server (downstream) service.")),
	mcp.WithString("start", mcp.Description("Start time for the query.")),
	mcp.WithString("end", mcp.Description("End time for the query. Default is now.")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive step sizing.")),
)

var ServicesTopologyTool = NewTool(
	"query_services_topology",
	`Query the service topology from SkyWalking OAP.

- If service_ids is provided, returns the topology scoped to those specific services (getServicesTopology).
- Otherwise, returns the global topology across all services (getGlobalTopology), optionally filtered by layer.

Examples:
- {}: Global topology for the last 30 minutes
- {"layer": "GENERAL"}: Global topology for a specific layer
- {"service_ids": ["svc-id-1", "svc-id-2"], "start": "-1h"}: Topology for specific services`,
	queryServicesTopology,
	mcp.WithArray("service_ids",
		mcp.Description("List of service IDs to scope the topology. If empty, the global topology is returned."),
		mcp.WithStringItems(),
	),
	mcp.WithString("layer",
		mcp.Description("Layer to filter the global topology (e.g. GENERAL, MESH). Only used when service_ids is empty.")),
	mcp.WithString("start", mcp.Description("Start time for the query.")),
	mcp.WithString("end", mcp.Description("End time for the query. Default is now.")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive step sizing.")),
)
