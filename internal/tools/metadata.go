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
)

// AddMetadataTools registers metadata-related tools with the MCP server
func AddMetadataTools(s *server.MCPServer) {
	ListLayersTool.Register(s)
	ListServicesTool.Register(s)
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
