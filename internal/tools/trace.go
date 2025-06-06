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
	"fmt"

	"github.com/apache/skywalking-cli/pkg/graphql/trace"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	api "skywalking.apache.org/repo/goapi/query"
)

type TraceRequest struct {
	TraceID string `json:"trace_id"`
}

func searchTrace(ctx context.Context, req TraceRequest) (*api.Trace, error) {
	traces, err := trace.Trace(ctx, req.TraceID)
	if err != nil {
		return nil, fmt.Errorf("search trace %v failed: %w", req.TraceID, err)
	}
	return &traces, nil
}

func AddTraceTools(mcp *server.MCPServer) {
	SearchTraceTool.Register(mcp)
}

var SearchTraceTool = NewTool[TraceRequest, *api.Trace](
	"search_trace_by_trace_id",
	"Search for traces by a single TraceId",
	searchTrace,
	mcp.WithTitleAnnotation("Search a trace by TraceId"),
	mcp.WithString("trace_id", mcp.Required(),
		mcp.Description("The TraceId to search for")),
)
