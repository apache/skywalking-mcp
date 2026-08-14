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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	api "skywalking.apache.org/repo/goapi/query"

	swevent "github.com/apache/skywalking-cli/pkg/graphql/event"
)

// AddEventTools registers event-related tools with the MCP server
func AddEventTools(s *mcp.Server) {
	mcp.AddTool(s, eventQueryTool(), queryEvents)
}

const orderASC = "ASC"

type EventQueryRequest struct {
	UUID            string `json:"uuid,omitempty" jsonschema:"Filter by event UUID."`
	Service         string `json:"service,omitempty" jsonschema:"Service name to filter events."`
	ServiceInstance string `json:"service_instance,omitempty" jsonschema:"Service instance name to filter events."`
	Endpoint        string `json:"endpoint,omitempty" jsonschema:"Endpoint name to filter events."`
	Name            string `json:"name,omitempty" jsonschema:"Event name to filter."`
	Type            string `json:"type,omitempty" jsonschema:"Event type: Normal or Error."`
	Layer           string `json:"layer,omitempty" jsonschema:"Layer to filter events."`
	Start           string `json:"start,omitempty" jsonschema:"Start time for the query."`
	End             string `json:"end,omitempty" jsonschema:"End time for the query. Default is now."`
	Step            string `json:"step,omitempty" jsonschema:"Time step granularity. If not specified, uses adaptive step sizing."`
	Order           string `json:"order,omitempty" jsonschema:"Order events by time: ASC (oldest first) or DES (newest first, default)."`
	PageNum         int    `json:"page_num,omitempty" jsonschema:"Page number, default 1."`
	PageSize        int    `json:"page_size,omitempty" jsonschema:"Page size, default 15."`
}

func buildEventQueryCondition(req *EventQueryRequest, timeCtx TimeContext) *api.EventQueryCondition {
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	cond := &api.EventQueryCondition{
		Time:   &duration,
		Paging: BuildPagination(req.PageNum, req.PageSize),
	}

	if req.UUID != "" {
		cond.UUID = &req.UUID
	}
	if req.Service != "" || req.ServiceInstance != "" || req.Endpoint != "" {
		src := &api.SourceInput{}
		if req.Service != "" {
			src.Service = &req.Service
		}
		if req.ServiceInstance != "" {
			src.ServiceInstance = &req.ServiceInstance
		}
		if req.Endpoint != "" {
			src.Endpoint = &req.Endpoint
		}
		cond.Source = src
	}
	if req.Name != "" {
		cond.Name = &req.Name
	}
	if req.Type != "" {
		t := api.EventType(req.Type)
		cond.Type = &t
	}
	if req.Layer != "" {
		cond.Layer = &req.Layer
	}

	order := api.OrderDes
	if req.Order == orderASC {
		order = api.OrderAsc
	}
	cond.Order = &order

	return cond
}

func queryEvents(ctx context.Context, _ *mcp.CallToolRequest, req EventQueryRequest) (*mcp.CallToolResult, any, error) {
	timeCtx := GetTimeContext(ctx)
	cond := buildEventQueryCondition(&req, timeCtx)

	events, err := swevent.Events(ctx, cond)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to query events: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(events)
	if err != nil {
		return ResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

func eventQueryTool() *mcp.Tool {
	schema := InferSchema[EventQueryRequest]()
	WithEnum(schema, "type", "Normal", "Error")
	WithEnum(schema, "step", stepEnum...)
	WithEnum(schema, "order", orderASC, "DES")

	return &mcp.Tool{
		Name: "query_events",
		Description: `Query events from SkyWalking OAP.
Events record changes or incidents on a service, instance, or endpoint (e.g. deployments, restarts, scaling).

Examples:
- {"service": "Your_ApplicationName", "start": "-1h"}: Recent events for a service
- {"type": "Error", "start": "-30m"}: Error events in the last 30 minutes
- {"service": "Your_ApplicationName", "type": "Normal"}: Normal events for a service`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "query_events", IdempotentHint: true},
	}
}
