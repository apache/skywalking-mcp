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

func performanceAnalysisHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	serviceName := args["service_name"]
	start := args["start"]
	end := args["end"]

	if start == "" {
		start = defaultDuration
	}
	if end == "" {
		end = defaultEnd
	}

	// Use the dynamic tool instructions
	toolInstructions := generateToolInstructions("performance_analysis")

	prompt := fmt.Sprintf(`Please analyze the performance of service '%s' for the time range start="%s", end="%s".

%s

**Analysis Required:**

Use start="%[2]s", end="%[3]s" on every tool call below.

**Response Time Analysis**
- Use execute_mqe_expression with expression="service_resp_time", start="%[2]s", end="%[3]s"
- Use execute_mqe_expression with expression="service_percentile{p='50,75,90,95,99'}", start="%[2]s", end="%[3]s"
- Identify trends and anomalies

**Success Rate and SLA**
- Use execute_mqe_expression with expression="service_sla / 100", start="%[2]s", end="%[3]s"
- Use execute_mqe_expression with expression="service_apdex / 10000", start="%[2]s", end="%[3]s"
- Track SLA compliance over time

**Traffic Analysis**
- Use execute_mqe_expression with expression="service_cpm", start="%[2]s", end="%[3]s"
- Identify traffic patterns and peak periods

**Error Analysis**
- Use query_traces with trace_state="error", start="%[2]s", end="%[3]s" to find error traces
- Identify most common error types and affected endpoints

**Performance Bottlenecks**
- Use execute_mqe_expression with expression="top_n(endpoint_resp_time, 5, DES)", start="%[2]s", end="%[3]s"
- Use execute_mqe_expression with expression="top_n(endpoint_cpm, 5, DES)", start="%[2]s", end="%[3]s"

Please provide actionable insights and specific recommendations based on the data.`, serviceName, start, end, toolInstructions)

	return &mcp.GetPromptResult{
		Description: "Performance analysis using SkyWalking tools",
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

func mqeQueryBuilderHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	queryType := args["query_type"]
	metrics := args["metrics"]
	conditions := args["conditions"]

	// Use the dynamic tool instructions
	toolInstructions := generateToolInstructions("mqe_query_building")

	prompt := fmt.Sprintf(`Help me build an MQE (Metrics Query Expression) for the following requirement:

Query Type: %s
Metrics: %s
Additional Conditions: %s

%s

**MQE Building Process:**

**Step-by-step approach:**
- Explain the MQE syntax for this use case
- Provide the complete MQE expression
- Show example usage with different parameters
- Explain what each part of the expression does
- Suggest variations for different scenarios

If there are multiple ways to achieve this, please show alternatives with pros and cons.`,
		queryType, metrics, conditions, toolInstructions)

	return &mcp.GetPromptResult{
		Description: "MQE query building assistance",
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

func compareServicesHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	services := args["services"]
	metrics := args["metrics"]
	start := args["start"]
	end := args["end"]

	if metrics == "" {
		metrics = allMetrics
	}
	if start == "" {
		start = defaultDuration
	}
	if end == "" {
		end = defaultEnd
	}

	prompt := fmt.Sprintf(`Please compare the following services: %s

Time Range: start="%s", end="%s"
Metrics to Compare: %s

Use start="%[2]s", end="%[3]s" on every execute_mqe_expression call.

Comparison should include:

1. **Performance Comparison**
   - Response time comparison (average and percentiles)
   - Throughput (CPM) comparison
   - Success rate (SLA) comparison

2. **Resource Utilization**
   - CPU and memory usage if available
   - Connection pool usage

3. **Error Patterns**
   - Error rate comparison
   - Types of errors by service

4. **Dependency Impact**
   - How each service affects others
   - Cascade failure risks

5. **Relative Performance**
   - Which service is the bottleneck
   - Performance ratios
   - Efficiency metrics

Please present the comparison in a clear, tabular format where possible, and highlight significant differences.`,
		services, start, end, metrics)

	return &mcp.GetPromptResult{
		Description: "Service comparison analysis",
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

func topServicesHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	metricName := args["metric_name"]
	topN := args["top_n"]
	order := args["order"]

	if topN == "" {
		topN = "10"
	}
	if order == "" {
		order = "DES"
	}

	prompt := fmt.Sprintf(`Find top services using execute_mqe_expression tool:

**Tool Configuration:**
- execute_mqe_expression with expression: "top_n(%s, %s, %s)"

**Analysis Focus:**

**Service Ranking**
- Get top %s services by %s
- Compare values against baseline
- Identify outliers or anomalies

**Performance Insights**
- For CPM metrics: Find busiest services
- For response time: Find slowest services
- For SLA: Find services with issues

**Actionable Recommendations**
- Services needing immediate attention
- Capacity planning insights
- Performance optimization targets

**Follow-up Analysis**
- Use query_traces for error investigation
- Use execute_mqe_expression for additional metric analysis

Provide ranked results with specific recommendations.`, metricName, topN, order, topN, metricName)

	return &mcp.GetPromptResult{
		Description: "Top services analysis",
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
