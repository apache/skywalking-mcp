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
	"github.com/modelcontextprotocol/go-sdk/mcp"
	api "skywalking.apache.org/repo/goapi/query"

	"github.com/apache/skywalking-cli/pkg/graphql/client"
	"github.com/apache/skywalking-cli/pkg/graphql/dependency"
)

// AddTopologyTools registers topology-related tools with the MCP server
func AddTopologyTools(s *mcp.Server) {
	mcp.AddTool(s, servicesTopologyTool(), queryServicesTopology)
	mcp.AddTool(s, instancesTopologyTool(), queryInstancesTopology)
	mcp.AddTool(s, endpointsTopologyTool(), queryEndpointsTopology)
	mcp.AddTool(s, processesTopologyTool(), queryProcessesTopology)
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
	ServiceIDs []string `json:"service_ids,omitempty" jsonschema:"List of service IDs to scope the topology. If empty, the global topology is returned."`
	Layer      string   `json:"layer,omitempty"`
	Start      string   `json:"start,omitempty" jsonschema:"Start time for the query."`
	End        string   `json:"end,omitempty" jsonschema:"End time for the query. Default is now."`
	Step       string   `json:"step,omitempty" jsonschema:"Time step granularity. If not specified, uses adaptive step sizing."`
}

func queryServicesTopology(ctx context.Context, _ *mcp.CallToolRequest, req ServicesTopologyRequest) (*mcp.CallToolResult, any, error) {
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
		return ResultError(fmt.Sprintf("failed to query topology: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return ResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

type InstancesTopologyRequest struct {
	ClientServiceID string `json:"client_service_id,omitempty" jsonschema:"The ID of the client (upstream) service."`
	ServerServiceID string `json:"server_service_id,omitempty" jsonschema:"The ID of the server (downstream) service."`
	Start           string `json:"start,omitempty" jsonschema:"Start time for the query."`
	End             string `json:"end,omitempty" jsonschema:"End time for the query. Default is now."`
	Step            string `json:"step,omitempty" jsonschema:"Time step granularity. If not specified, uses adaptive step sizing."`
}

func queryInstancesTopology(ctx context.Context, _ *mcp.CallToolRequest, req InstancesTopologyRequest) (*mcp.CallToolResult, any, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	topology, err := dependency.InstanceTopology(ctx, req.ClientServiceID, req.ServerServiceID, duration)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to query instances topology: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return ResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

type EndpointsTopologyRequest struct {
	EndpointID string `json:"endpoint_id,omitempty" jsonschema:"The ID of the endpoint to query dependencies for."`
	Start      string `json:"start,omitempty" jsonschema:"Start time for the query."`
	End        string `json:"end,omitempty" jsonschema:"End time for the query. Default is now."`
	Step       string `json:"step,omitempty" jsonschema:"Time step granularity. If not specified, uses adaptive step sizing."`
}

func queryEndpointsTopology(ctx context.Context, _ *mcp.CallToolRequest, req EndpointsTopologyRequest) (*mcp.CallToolResult, any, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	topology, err := dependency.EndpointDependency(ctx, req.EndpointID, duration)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to query endpoints topology: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return ResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

type ProcessesTopologyRequest struct {
	ServiceInstanceID string `json:"service_instance_id,omitempty" jsonschema:"The ID of the service instance to query process topology for."`
	Start             string `json:"start,omitempty" jsonschema:"Start time for the query."`
	End               string `json:"end,omitempty" jsonschema:"End time for the query. Default is now."`
	Step              string `json:"step,omitempty" jsonschema:"Time step granularity. If not specified, uses adaptive step sizing."`
}

func queryProcessesTopology(ctx context.Context, _ *mcp.CallToolRequest, req ProcessesTopologyRequest) (*mcp.CallToolResult, any, error) {
	timeCtx := GetTimeContext(ctx)
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	topology, err := dependency.ProcessTopology(ctx, req.ServiceInstanceID, duration)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to query processes topology: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(topology)
	if err != nil {
		return ResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

func processesTopologyTool() *mcp.Tool {
	schema := InferSchema[ProcessesTopologyRequest]()
	WithRequired(schema, "service_instance_id")
	WithEnum(schema, "step", stepEnum...)

	return &mcp.Tool{
		Name: "query_processes_topology",
		Description: `Query the process topology for a given service instance from SkyWalking OAP (getProcessTopology).

Returns the topology of processes running within the given service instance, including process nodes and the calls between them.

Examples:
- {"service_instance_id": "instance-id-1"}: Process topology for the last 30 minutes
- {"service_instance_id": "instance-id-1", "start": "-1h"}: Process topology for the last hour`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "query_processes_topology", IdempotentHint: true},
	}
}

func endpointsTopologyTool() *mcp.Tool {
	schema := InferSchema[EndpointsTopologyRequest]()
	WithRequired(schema, "endpoint_id")
	WithEnum(schema, "step", stepEnum...)

	return &mcp.Tool{
		Name: "query_endpoints_topology",
		Description: `Query the endpoint dependency topology for a given endpoint from SkyWalking OAP (getEndpointDependencies).

Returns the topology of endpoints that the given endpoint depends on or is depended upon by, including endpoint nodes and the calls between them.

Examples:
- {"endpoint_id": "ep-id-1"}: Endpoint topology for the last 30 minutes
- {"endpoint_id": "ep-id-1", "start": "-1h"}: Endpoint topology for the last hour`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "query_endpoints_topology", IdempotentHint: true},
	}
}

func instancesTopologyTool() *mcp.Tool {
	schema := InferSchema[InstancesTopologyRequest]()
	WithRequired(schema, "client_service_id", "server_service_id")
	WithEnum(schema, "step", stepEnum...)

	return &mcp.Tool{
		Name: "query_instances_topology",
		Description: `Query the service instance topology between two services from SkyWalking OAP (getServiceInstanceTopology).

Returns the topology of service instances for the given client and server services, including instance nodes and the calls between them.

Examples:
- {"client_service_id": "svc-id-1", "server_service_id": "svc-id-2"}: Instance topology for the last 30 minutes
- {"client_service_id": "svc-id-1", "server_service_id": "svc-id-2", "start": "-1h"}: Instance topology for the last hour`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "query_instances_topology", IdempotentHint: true},
	}
}

func servicesTopologyTool() *mcp.Tool {
	schema := InferSchema[ServicesTopologyRequest]()
	WithEnum(schema, "step", stepEnum...)
	WithDescriptions(schema, map[string]string{
		"layer": "Layer to filter the global topology (e.g. GENERAL, MESH). " +
			"Only used when service_ids is empty.",
	})

	return &mcp.Tool{
		Name: "query_services_topology",
		Description: `Query the service topology from SkyWalking OAP.

- If service_ids is provided, returns the topology scoped to those specific services (getServicesTopology).
- Otherwise, returns the global topology across all services (getGlobalTopology), optionally filtered by layer.

Examples:
- {}: Global topology for the last 30 minutes
- {"layer": "GENERAL"}: Global topology for a specific layer
- {"service_ids": ["svc-id-1", "svc-id-2"], "start": "-1h"}: Topology for specific services`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "query_services_topology", IdempotentHint: true},
	}
}
