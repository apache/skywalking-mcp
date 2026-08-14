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

	swalarm "github.com/apache/skywalking-cli/pkg/graphql/alarm"
)

// AddAlarmTools registers alarm-related tools with the MCP server
func AddAlarmTools(s *mcp.Server) {
	mcp.AddTool(s, alarmQueryTool(), queryAlarms)
}

type AlarmTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AlarmQueryRequest struct {
	Scope    string     `json:"scope,omitempty" jsonschema:"Scope to filter alarms."`
	Keyword  string     `json:"keyword,omitempty" jsonschema:"Keyword to filter alarm messages."`
	Tags     []AlarmTag `json:"tags,omitempty" jsonschema:"Array of alarm tags to filter by, each with key and value."`
	Start    string     `json:"start,omitempty" jsonschema:"Start time for the query."`
	End      string     `json:"end,omitempty" jsonschema:"End time for the query. Default is now."`
	Step     string     `json:"step,omitempty" jsonschema:"Time step granularity. If not specified, uses adaptive step sizing."`
	PageNum  int        `json:"page_num,omitempty" jsonschema:"Page number, default 1."`
	PageSize int        `json:"page_size,omitempty" jsonschema:"Page size, default 15."`
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

func queryAlarms(ctx context.Context, _ *mcp.CallToolRequest, req AlarmQueryRequest) (*mcp.CallToolResult, any, error) {
	timeCtx := GetTimeContext(ctx)
	cond := buildAlarmQueryCondition(&req, timeCtx)

	alarms, err := swalarm.Alarms(ctx, cond)
	if err != nil {
		return ResultError(fmt.Sprintf("failed to query alarms: %v", err)), nil, nil
	}

	jsonBytes, err := json.Marshal(alarms)
	if err != nil {
		return ResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil, nil
	}
	return ResultText(string(jsonBytes)), nil, nil
}

func alarmQueryTool() *mcp.Tool {
	schema := InferSchema[AlarmQueryRequest]()
	WithEnum(schema, "scope", "All", "Service", "ServiceInstance", "Endpoint", "Process",
		"ServiceRelation", "ServiceInstanceRelation", "EndpointRelation", "ProcessRelation")
	WithEnum(schema, "step", stepEnum...)

	return &mcp.Tool{
		Name: "query_alarms",
		Description: `Query alarms from SkyWalking OAP. Alarms are triggered when metrics breach configured thresholds.

Examples:
- {"start": "-1h"}: All alarms in the last hour
- {"scope": "Service", "start": "-30m"}: Service-level alarms in the last 30 minutes
- {"keyword": "timeout", "start": "-1h"}: Alarms whose message contains "timeout"
- {"tags": [{"key": "level", "value": "critical"}], "start": "-1h"}: Alarms with a specific tag`,
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{Title: "query_alarms", IdempotentHint: true},
	}
}
