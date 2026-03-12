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

package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func traceInvestigationHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	serviceID := args["service_id"]
	traceState := args["trace_state"]
	start := args["start"]
	end := args["end"]

	if start == "" {
		start = defaultDuration
	}
	if end == "" {
		end = defaultEnd
	}
	if traceState == "" {
		traceState = "all"
	}

	// Use the dynamic tool instructions
	toolInstructions := generateToolInstructions("trace_investigation")

	prompt := fmt.Sprintf(`Investigate traces with filters: service_id="%s", trace_state="%s", start="%s", end="%s".

%s

**Analysis Steps:**

**Find Problematic Traces**
- First use query_traces with start="%[3]s", end="%[4]s", view="summary" to get overview
- Look for patterns in error traces, slow traces, or anomalies
- Note trace IDs that need deeper investigation

**Deep Dive on Specific Traces**
- Use query_traces with the identified trace_id
- Start with view="summary" for quick insights
- Use view="full" for complete span analysis
- Use view="errors_only" if focusing on errors

**Performance Analysis**
- Look for traces with high duration using min_trace_duration filter
- Identify bottlenecks in span timings
- Check for cascading delays

**Error Pattern Analysis**
- Use query_traces with trace_state="error"
- Group errors by type and service
- Identify error propagation paths

Provide specific findings and actionable recommendations.`, serviceID, traceState, start, end, toolInstructions)

	return &mcp.GetPromptResult{
		Description: "Trace investigation using query tools",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}

func logAnalysisHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	serviceID := args["service_id"]
	logLevel := args["log_level"]
	start := args["start"]
	end := args["end"]

	if start == "" {
		start = defaultDuration
	}
	if end == "" {
		end = defaultEnd
	}
	if logLevel == "" {
		logLevel = "ERROR"
	}

	prompt := fmt.Sprintf(`Analyze service logs using the query_logs tool:

**Tool Configuration:**
- query_logs with following parameters:
  - service_id: "%s" (if specified)
  - tags: [{"key": "level", "value": "%s"}] for log level filtering
  - start: "%s", end: "%s" for time range
  - cold: true if historical data needed

**Analysis Steps:**

**Log Pattern Analysis**
- Use query_logs to get recent logs for the service
- Filter by log level (ERROR, WARN, INFO)
- Look for recurring error patterns
- Identify frequency of different log types

**Error Investigation**
- Focus on ERROR level logs first
- Group similar error messages
- Check for correlation with trace IDs
- Look for timestamp patterns

**Performance Correlation**
- Compare log timestamps with performance issues
- Look for resource exhaustion indicators
- Check for timeout or connection errors

**Troubleshooting Workflow**
- Start with ERROR logs in the specified time range
- Use trace_id from logs to get detailed trace analysis
- Cross-reference with metrics for full picture

Provide specific log analysis findings and recommendations.`, serviceID, logLevel, start, end)

	return &mcp.GetPromptResult{
		Description: "Log analysis using query_logs tool",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}

func traceDeepDiveHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	traceID := args["trace_id"]
	view := args["view"]

	if view == "" {
		view = "summary"
	}

	prompt := fmt.Sprintf(`Perform deep dive analysis of trace %s:

**Primary Analysis:**
- Use query_traces with trace_id: "%s" and view: "%s"
- Start with summary view for quick insights
- Use full view for complete span analysis
- Use errors_only view if trace has errors

**Trace Structure Analysis**
- Service call flow and dependencies
- Span duration breakdown
- Critical path identification
- Parallel vs sequential operations

**Performance Investigation**
- Identify bottleneck spans
- Database query performance
- External API call latency
- Resource wait times

**Error Analysis** (if applicable)
- Error location and propagation
- Root cause identification
- Impact assessment

**Optimization Opportunities**
- Redundant operations
- Caching possibilities
- Parallel processing potential
- Database query optimization

Provide detailed trace analysis with specific optimization recommendations.`, traceID, traceID, view)

	return &mcp.GetPromptResult{
		Description: "Deep dive trace analysis",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: prompt,
				},
			},
		},
	}, nil
}
