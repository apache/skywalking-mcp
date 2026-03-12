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
	api "skywalking.apache.org/repo/goapi/query"

	swevent "github.com/apache/skywalking-cli/pkg/graphql/event"
)

// AddEventTools registers event-related tools with the MCP server
func AddEventTools(s *server.MCPServer) {
	EventQueryTool.Register(s)
}

const orderASC = "ASC"

type EventQueryRequest struct {
	UUID            string `json:"uuid,omitempty"`
	Service         string `json:"service,omitempty"`
	ServiceInstance string `json:"service_instance,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	Name            string `json:"name,omitempty"`
	Type            string `json:"type,omitempty"`
	Layer           string `json:"layer,omitempty"`
	Start           string `json:"start,omitempty"`
	End             string `json:"end,omitempty"`
	Step            string `json:"step,omitempty"`
	Order           string `json:"order,omitempty"`
	PageNum         int    `json:"page_num,omitempty"`
	PageSize        int    `json:"page_size,omitempty"`
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

func queryEvents(ctx context.Context, req *EventQueryRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	cond := buildEventQueryCondition(req, timeCtx)

	events, err := swevent.Events(ctx, cond)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query events: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(events)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

var EventQueryTool = NewTool(
	"query_events",
	`Query events from SkyWalking OAP. Events record changes or incidents on a service, instance, or endpoint (e.g. deployments, restarts, scaling).

Examples:
- {"service": "Your_ApplicationName", "start": "-1h"}: Recent events for a service
- {"type": "Error", "start": "-30m"}: Error events in the last 30 minutes
- {"service": "Your_ApplicationName", "type": "Normal"}: Normal events for a service`,
	queryEvents,
	mcp.WithString("uuid", mcp.Description("Filter by event UUID.")),
	mcp.WithString("service", mcp.Description("Service name to filter events.")),
	mcp.WithString("service_instance", mcp.Description("Service instance name to filter events.")),
	mcp.WithString("endpoint", mcp.Description("Endpoint name to filter events.")),
	mcp.WithString("name", mcp.Description("Event name to filter.")),
	mcp.WithString("type", mcp.Enum("Normal", "Error"),
		mcp.Description("Event type: Normal or Error.")),
	mcp.WithString("layer", mcp.Description("Layer to filter events.")),
	mcp.WithString("start", mcp.Description("Start time for the query.")),
	mcp.WithString("end", mcp.Description("End time for the query. Default is now.")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive step sizing.")),
	mcp.WithString("order", mcp.Enum(orderASC, "DES"),
		mcp.Description("Order events by time: ASC (oldest first) or DES (newest first, default).")),
	mcp.WithNumber("page_num", mcp.Description("Page number, default 1.")),
	mcp.WithNumber("page_size", mcp.Description("Page size, default 15.")),
)
