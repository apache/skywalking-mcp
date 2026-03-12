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

func generateDurationHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	timeRange := request.Params.Arguments["time_range"]

	prompt := fmt.Sprintf(`Convert the following time range description into a duration object with "start" and "end" fields.

Time range: "%s"

Rules:
- "start" and "end" must be strings in one of these formats:
  - Relative: "-30m" (30 minutes ago), "-1h" (1 hour ago), "-7d" (7 days ago)
  - Absolute: "2024-01-01 12:00:00" (YYYY-MM-DD HH:MM:SS)
- If the end of the range is the current time, set "end" to "now"
- Relative values are always negative (e.g. "-1h", not "1h")

Output only a JSON object, for example:
{"start": "-1h", "end": "now"}

This duration can be passed directly to tools such as list_instances, list_endpoints, and list_processes.`, timeRange)

	return &mcp.GetPromptResult{
		Description: "Generate a {start, end} duration object from a natural-language time range",
		Messages: []mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.TextContent{Type: "text", Text: prompt},
			},
		},
	}, nil
}

func exploreServiceTopologyHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	layer := args["layer"]
	start := args["start"]
	end := args["end"]

	if end == "" {
		end = defaultEnd
	}

	prompt := fmt.Sprintf(`Explore the service topology of layer "%s" within the time range from "%s" to "%s".

**Workflow:**

**Step 1 – Discover services**
- Use list_services with layer="%s" to get all services in this layer
- Note the id of each service for the next steps

**Step 2 – List instances per service**
- For each service of interest, use list_instances with:
  - service_id: <id from step 1>
  - start: "%s"
  - end: "%s"
- Review instance names, languages, and attributes

**Step 3 – List endpoints per service**
- Use list_endpoints with:
  - service_id: <id from step 1>
  - start: "%s"
  - end: "%s"
- Note endpoint ids for use in metrics or trace queries

**Step 4 – List processes per instance**
- For each instance of interest, use list_processes with:
  - instance_id: <id from step 2>
  - start: "%s"
  - end: "%s"
- Review process names, detect types, and labels

**Summary to provide:**
- Total number of services, instances, endpoints, and processes found
- Any notable attributes or labels worth highlighting
- Suggested follow-up queries (e.g. metrics, traces, logs) for specific services or instances`,
		layer, start, end,
		layer,
		start, end,
		start, end,
		start, end)

	return &mcp.GetPromptResult{
		Description: "Service topology exploration",
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

func exploreMetricsHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	pattern := args["pattern"]
	showExamples := args["show_examples"]

	if pattern == "" {
		pattern = ".*" // match all metrics
	}

	// Use the dynamic tool instructions
	toolInstructions := generateToolInstructions("metrics_exploration")

	prompt := fmt.Sprintf(`Explore available metrics with pattern: "%s".

%s

**Exploration Workflow:**

**Discover Metrics**
- Use list_mqe_metrics to get all available metrics
- Filter by pattern if specified
- Review metric names and types

**Understand Metric Types**
- For each interesting metric, use get_mqe_metric_type
- REGULAR_VALUE: Direct arithmetic operations
- LABELED_VALUE: Requires label selectors
- SAMPLED_RECORD: Complex record-based metrics

**Usage Examples** (if show_examples is "%s"):
- REGULAR_VALUE: service_cpm, service_sla * 100
- LABELED_VALUE: service_percentile{p='50,75,90,95,99'}
- Complex: avg(service_cpm), top_n(service_resp_time, 10, des)

**Metric Categories:**
- Service metrics: service_sla, service_cpm, service_resp_time
- Instance metrics: service_instance_*
- Endpoint metrics: endpoint_*
- Relation metrics: service_relation_*
- Infrastructure metrics: service_cpu, service_memory

**Best Practices:**
- Check metric type before using in expressions
- Use appropriate label selectors for LABELED_VALUE
- Combine metrics for comprehensive analysis
- Use aggregation functions for trend analysis

Provide a comprehensive guide to available metrics and their usage.`, pattern, toolInstructions, showExamples)

	return &mcp.GetPromptResult{
		Description: "Metrics exploration guide",
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
