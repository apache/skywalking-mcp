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

	swalarm "github.com/apache/skywalking-cli/pkg/graphql/alarm"
)

// AddAlarmTools registers alarm-related tools with the MCP server
func AddAlarmTools(s *server.MCPServer) {
	AlarmQueryTool.Register(s)
}

type AlarmTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AlarmQueryRequest struct {
	Scope    string     `json:"scope,omitempty"`
	Keyword  string     `json:"keyword,omitempty"`
	Tags     []AlarmTag `json:"tags,omitempty"`
	Start    string     `json:"start,omitempty"`
	End      string     `json:"end,omitempty"`
	Step     string     `json:"step,omitempty"`
	PageNum  int        `json:"page_num,omitempty"`
	PageSize int        `json:"page_size,omitempty"`
}

func buildAlarmQueryCondition(req *AlarmQueryRequest, timeCtx TimeContext) *swalarm.ListAlarmCondition {
	duration := BuildDurationWithContext(req.Start, req.End, req.Step, false, DefaultDuration, timeCtx)

	var tags []*api.AlarmTag
	for _, t := range req.Tags {
		v := t.Value
		tags = append(tags, &api.AlarmTag{Key: t.Key, Value: &v})
	}

	cond := &swalarm.ListAlarmCondition{
		Duration: &duration,
		Keyword:  req.Keyword,
		Tags:     tags,
		Paging:   BuildPagination(req.PageNum, req.PageSize),
	}

	if req.Scope != "" {
		cond.Scope = api.Scope(req.Scope)
	}

	return cond
}

func queryAlarms(ctx context.Context, req *AlarmQueryRequest) (*mcp.CallToolResult, error) {
	timeCtx := GetTimeContext(ctx)
	cond := buildAlarmQueryCondition(req, timeCtx)

	alarms, err := swalarm.Alarms(ctx, cond)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to query alarms: %v", err)), nil
	}

	jsonBytes, err := json.Marshal(alarms)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

var AlarmQueryTool = NewTool(
	"query_alarms",
	`Query alarms from SkyWalking OAP. Alarms are triggered when metrics breach configured thresholds.

Examples:
- {"start": "-1h"}: All alarms in the last hour
- {"scope": "Service", "start": "-30m"}: Service-level alarms in the last 30 minutes
- {"keyword": "timeout", "start": "-1h"}: Alarms whose message contains "timeout"
- {"tags": [{"key": "level", "value": "critical"}], "start": "-1h"}: Alarms with a specific tag`,
	queryAlarms,
	mcp.WithString("scope",
		mcp.Enum("All", "Service", "ServiceInstance", "Endpoint", "Process",
			"ServiceRelation", "ServiceInstanceRelation", "EndpointRelation", "ProcessRelation"),
		mcp.Description("Scope to filter alarms.")),
	mcp.WithString("keyword", mcp.Description("Keyword to filter alarm messages.")),
	mcp.WithArray("tags",
		mcp.Description("Array of alarm tags to filter by, each with key and value."),
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
		mcp.Description("Time step granularity. If not specified, uses adaptive step sizing.")),
	mcp.WithNumber("page_num", mcp.Description("Page number, default 1.")),
	mcp.WithNumber("page_size", mcp.Description("Page size, default 15.")),
)
