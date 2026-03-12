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

	swlog "github.com/apache/skywalking-cli/pkg/graphql/log"
)

// AddLogTools registers log-related tools with the MCP server
func AddLogTools(mcp *server.MCPServer) {
	LogQueryTool.Register(mcp)
}

type LogTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LogQueryRequest struct {
	ServiceID         string   `json:"service_id,omitempty"`
	ServiceInstanceID string   `json:"service_instance_id,omitempty"`
	EndpointID        string   `json:"endpoint_id,omitempty"`
	TraceID           string   `json:"trace_id,omitempty"`
	SegmentID         string   `json:"segment_id,omitempty"`
	SpanID            *int     `json:"span_id,omitempty"`
	Tags              []LogTag `json:"tags,omitempty"`
	Start             string   `json:"start,omitempty"`
	End               string   `json:"end,omitempty"`
	Step              string   `json:"step,omitempty"`
	Cold              bool     `json:"cold,omitempty"`
	PageNum           int      `json:"page_num,omitempty"`
	PageSize          int      `json:"page_size,omitempty"`
	QueryOrder        string   `json:"query_order,omitempty"`
}

// buildLogQueryCondition builds the log query condition from request parameters
func buildLogQueryCondition(req *LogQueryRequest, timeCtx TimeContext) *api.LogQueryCondition {
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, req.Cold, DefaultDuration, timeCtx)

	var tags []*api.LogTag
	for _, t := range req.Tags {
		v := t.Value
		tags = append(tags, &api.LogTag{Key: t.Key, Value: &v})
	}

	paging := BuildPagination(req.PageNum, req.PageSize)

	order := api.OrderDes
	if req.QueryOrder == "ASC" {
		order = api.OrderAsc
	}

	cond := &api.LogQueryCondition{
		ServiceID:         &req.ServiceID,
		ServiceInstanceID: &req.ServiceInstanceID,
		EndpointID:        &req.EndpointID,
		QueryDuration:     &duration,
		Paging:            paging,
		QueryOrder:        &order,
	}

	if req.TraceID != "" || req.SegmentID != "" || req.SpanID != nil {
		traceScope := &api.TraceScopeCondition{}

		if req.TraceID != "" {
			traceScope.TraceID = req.TraceID
		}

		if req.SegmentID != "" {
			// Only set SegmentID when it is actually provided to avoid filtering by an empty segment.
			traceScope.SegmentID = &req.SegmentID
		}

		if req.SpanID != nil {
			traceScope.SpanID = req.SpanID
		}

		cond.RelatedTrace = traceScope
	}

	if len(tags) > 0 {
		cond.Tags = tags
	}
	return cond
}

// queryLogs queries logs from SkyWalking OAP
func queryLogs(ctx context.Context, req *LogQueryRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	cond := buildLogQueryCondition(req, timeCtx)

	logs, err := swlog.Logs(ctx, cond)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query logs: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(logs)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

var LogQueryTool = NewTool(
	"query_logs",
	`Query logs from SkyWalking OAP with flexible filters.

Workflow:
1. Use this tool to find logs matching specific criteria
2. Specify one or more query conditions to narrow down results
3. Use start/end to limit the time range for the search
4. Supports filtering by service, instance, endpoint, trace, tags, and time
5. Supports cold storage query and pagination

Examples:
- {"service_id": "Your_ApplicationName", "start": "2024-06-01 12:00:00", "end": "2024-06-01 13:00:00"}: Query logs for a service in a time range
- {"trace_id": "abc123..."}: Query logs related to a specific trace
- {"trace_id": "abc123...", "segment_id": "seg456...", "span_id": 0}: Query logs for a specific span
- {"tags": [{"key": "level", "value": "ERROR"}], "cold": true}: Query error logs from cold storage`,
	queryLogs,
	mcp.WithString("service_id", mcp.Description("Service ID to filter logs.")),
	mcp.WithString("service_instance_id", mcp.Description("Service instance ID to filter logs.")),
	mcp.WithString("endpoint_id", mcp.Description("Endpoint ID to filter logs.")),
	mcp.WithString("trace_id", mcp.Description("Related trace ID to filter logs by trace scope.")),
	mcp.WithString("segment_id", mcp.Description("Related segment ID to narrow logs to a specific segment within a trace.")),
	mcp.WithNumber("span_id", mcp.Description("Related span ID to narrow logs to a specific span within a segment.")),
	mcp.WithArray("tags",
		mcp.Description("Array of log tags, each with key and value."),
		mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":   map[string]any{"type": "string"},
				"value": map[string]any{"type": "string"},
			},
			"required": []string{"key", "value"},
		}),
	),
	mcp.WithString("start", mcp.Description("Start time for the query.")),
	mcp.WithString("end", mcp.Description("End time for the query. Default is now.")),
	mcp.WithString("step", mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity: SECOND, MINUTE, HOUR, DAY. "+
			"If not specified, uses adaptive step sizing: "+
			"SECOND (<1h), MINUTE (1h-24h), HOUR (1d-7d), DAY (>7d)")),
	mcp.WithBoolean("cold", mcp.Description("Whether to query from cold-stage storage.")),
	mcp.WithNumber("page_num", mcp.Description("Page number, default 1.")),
	mcp.WithNumber("page_size", mcp.Description("Page size, default 15.")),
	mcp.WithString("query_order", mcp.Enum("ASC", "DES"),
		mcp.Description("Order logs by timestamp: ASC (oldest first) or DES (newest first, default).")),
)
