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
	"github.com/mark3labs/mcp-go/server"
)

// Constants for common values
const (
	defaultDuration = "-1h"
	allMetrics      = "all"
)

// Tool capability mapping for different analysis types
var toolCapabilities = map[string][]string{
	"performance_analysis": {
		"query_single_metrics",
		"query_top_n_metrics",
		"execute_mqe_expression",
	},
	"trace_investigation": {
		"query_traces",
		"get_trace_details",
		"get_cold_trace_details",
	},
	"log_analysis": {
		"query_logs",
	},
	"mqe_query_building": {
		"execute_mqe_expression",
		"list_mqe_metrics",
		"get_mqe_metric_type",
	},
	"service_comparison": {
		"query_single_metrics",
		"query_top_n_metrics",
		"execute_mqe_expression",
	},
	"metrics_exploration": {
		"list_mqe_metrics",
		"get_mqe_metric_type",
	},
}

// AddSkyWalkingPrompts registers all SkyWalking-related prompts
func AddSkyWalkingPrompts(s *server.MCPServer) {
	addCoreAnalysisPrompts(s)
	addTraceAnalysisPrompts(s)
	addUtilityPrompts(s)
}

func addCoreAnalysisPrompts(s *server.MCPServer) {
	// Performance Analysis Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "analyze-performance",
		Description: "Analyze service performance using metrics tools",
		Arguments: []mcp.PromptArgument{
			{Name: "service_name", Description: "The name of the service to analyze", Required: true},
			{Name: "duration", Description: "Time duration for analysis. Examples: -1h (past hour), -30m (past 30 minutes), " +
				"-7d (past 7 days), 1h (next hour), 24h (next 24 hours)", Required: false},
		},
	}, performanceAnalysisHandler)

	// Service Comparison Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "compare-services",
		Description: "Compare performance metrics between multiple services",
		Arguments: []mcp.PromptArgument{
			{Name: "services", Description: "Comma-separated list of service names to compare", Required: true},
			{Name: "metrics", Description: "Metrics to compare (response_time, sla, cpm, all)", Required: false},
			{Name: "time_range", Description: "Time range for comparison. Examples: -1h (last hour), -2h (last 2 hours), -1d (last day)", Required: false},
		},
	}, compareServicesHandler)

	// Top N Metrics Analysis
	s.AddPrompt(mcp.Prompt{
		Name:        "top-services",
		Description: "Find top N services by various metrics",
		Arguments: []mcp.PromptArgument{
			{Name: "metric_name", Description: "Metric to rank by (service_cpm, service_resp_time, service_sla)", Required: true},
			{Name: "top_n", Description: "Number of top services to return (default: 10)", Required: false},
			{Name: "order", Description: "Order direction (ASC, DES)", Required: false},
		},
	}, topServicesHandler)
}

func addTraceAnalysisPrompts(s *server.MCPServer) {
	// Trace Investigation Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "investigate-traces",
		Description: "Investigate traces for errors and performance issues",
		Arguments: []mcp.PromptArgument{
			{Name: "service_id", Description: "The service to investigate", Required: false},
			{Name: "trace_state", Description: "Filter by trace state (success, error, all)", Required: false},
			{Name: "duration", Description: "Time range to search. Examples: -1h (last hour), -30m (last 30 minutes). Default: -1h", Required: false},
		},
	}, traceInvestigationHandler)

	// Trace Deep Dive
	s.AddPrompt(mcp.Prompt{
		Name:        "trace-deep-dive",
		Description: "Deep dive analysis of a specific trace",
		Arguments: []mcp.PromptArgument{
			{Name: "trace_id", Description: "The trace ID to analyze", Required: true},
			{Name: "view", Description: "Analysis view (full, summary, errors_only)", Required: false},
			{Name: "check_cold_storage", Description: "Check cold storage if not found (true/false)", Required: false},
		},
	}, traceDeepDiveHandler)

	// Log Analysis Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "analyze-logs",
		Description: "Analyze service logs for errors and patterns",
		Arguments: []mcp.PromptArgument{
			{Name: "service_id", Description: "Service to analyze logs", Required: false},
			{Name: "log_level", Description: "Log level to filter (ERROR, WARN, INFO)", Required: false},
			{Name: "duration", Description: "Time range to analyze. Examples: -1h (last hour), -6h (last 6 hours). Default: -1h", Required: false},
		},
	}, logAnalysisHandler)
}

func addUtilityPrompts(s *server.MCPServer) {
	// MQE Query Builder Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "build-mqe-query",
		Description: "Help build MQE (Metrics Query Expression) for complex queries",
		Arguments: []mcp.PromptArgument{
			{Name: "query_type", Description: "Type of query (performance, comparison, trend, alert)", Required: true},
			{Name: "metrics", Description: "Comma-separated list of metrics to query", Required: true},
			{Name: "conditions", Description: "Additional conditions or filters", Required: false},
		},
	}, mqeQueryBuilderHandler)

	// MQE Metrics Explorer
	s.AddPrompt(mcp.Prompt{
		Name:        "explore-metrics",
		Description: "Explore available metrics and their types",
		Arguments: []mcp.PromptArgument{
			{Name: "pattern", Description: "Regex pattern to filter metrics", Required: false},
			{Name: "show_examples", Description: "Show usage examples for each metric (true/false)", Required: false},
		},
	}, exploreMetricsHandler)
}

// Analysis execution chains for different types of analysis
var analysisChains = map[string][]struct {
	Tool    string
	Purpose string
}{
	"performance_analysis": {
		{Tool: "query_single_metrics", Purpose: "Get basic metrics like CPM, SLA, response time"},
		{Tool: "execute_mqe_expression", Purpose: "Calculate derivatives like SLA percentage, percentiles"},
		{Tool: "query_top_n_metrics", Purpose: "Identify top endpoints by response time or traffic"},
		{Tool: "query_traces", Purpose: "Find error traces for deeper investigation"},
	},
	"trace_investigation": {
		{Tool: "query_traces", Purpose: "Search for traces with specific filters"},
		{Tool: "get_trace_details", Purpose: "Analyze individual traces in detail"},
		{Tool: "get_cold_trace_details", Purpose: "Check historical traces if not found in hot storage"},
	},
	"log_analysis": {
		{Tool: "query_logs", Purpose: "Search and analyze log entries with filters"},
	},
	"mqe_query_building": {
		{Tool: "list_mqe_metrics", Purpose: "Discover available metrics"},
		{Tool: "get_mqe_metric_type", Purpose: "Understand metric types and usage"},
		{Tool: "execute_mqe_expression", Purpose: "Test and execute the built expression"},
	},
}

// Helper function to generate tool usage instructions
func generateToolInstructions(analysisType string) string {
	tools := toolCapabilities[analysisType]
	chain := analysisChains[analysisType]

	if len(tools) == 0 {
		return "No specific tools defined for this analysis type."
	}

	instructions := "**Available Tools:**\n"
	for _, tool := range tools {
		instructions += fmt.Sprintf("- %s\n", tool)
	}

	if len(chain) > 0 {
		instructions += "\n**Recommended Analysis Workflow:**\n"
		for i, step := range chain {
			instructions += fmt.Sprintf("%d. %s: %s\n", i+1, step.Tool, step.Purpose)
		}
	}

	return instructions
}

// Handler implementations

func performanceAnalysisHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	serviceName := args["service_name"]
	duration := args["duration"]

	if duration == "" {
		duration = defaultDuration
	}

	// Use the dynamic tool instructions
	toolInstructions := generateToolInstructions("performance_analysis")

	prompt := fmt.Sprintf(`Please analyze the performance of service '%s' over the last %s.

%s

**Analysis Required:**

**Response Time Analysis**
- Use query_single_metrics with metrics_name="service_resp_time" to get average response time
- Use execute_mqe_expression with expression="service_percentile{p='50,75,90,95,99'}" to get percentiles
- Identify trends and anomalies

**Success Rate and SLA**
- Use execute_mqe_expression with expression="service_sla * 100" to get success rate percentage
- Use query_single_metrics with metrics_name="service_apdex" for user satisfaction score
- Track SLA compliance over time

**Traffic Analysis**
- Use query_single_metrics with metrics_name="service_cpm" to get calls per minute
- Identify traffic patterns and peak periods

**Error Analysis**
- Use query_traces with trace_state="error" to find error traces
- Identify most common error types and affected endpoints

**Performance Bottlenecks**
- Use query_top_n_metrics with metrics_name="endpoint_resp_time" and order="DES" to find slowest endpoints
- Use query_top_n_metrics with metrics_name="endpoint_cpm" to find high-traffic endpoints

Please provide actionable insights and specific recommendations based on the data.`, serviceName, duration, toolInstructions)

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
	timeRange := args["time_range"]

	if metrics == "" {
		metrics = allMetrics
	}
	if timeRange == "" {
		timeRange = defaultDuration
	}

	prompt := fmt.Sprintf(`Please compare the following services: %s

Time Range: %s
Metrics to Compare: %s

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
		services, timeRange, metrics)

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

func traceInvestigationHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	serviceID := args["service_id"]
	traceState := args["trace_state"]
	duration := args["duration"]

	if duration == "" {
		duration = defaultDuration
	}
	if traceState == "" {
		traceState = "all"
	}

	// Use the dynamic tool instructions
	toolInstructions := generateToolInstructions("trace_investigation")

	prompt := fmt.Sprintf(`Investigate traces with filters: service_id="%s", trace_state="%s", duration="%s".

%s

**Analysis Steps:**

**Find Problematic Traces**
- First use query_traces with view="summary" to get overview
- Look for patterns in error traces, slow traces, or anomalies
- Note trace IDs that need deeper investigation

**Deep Dive on Specific Traces**
- Use get_trace_details with identified trace_id
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

**Historical Investigation**
- If recent data shows no issues, use cold storage tools
- Use get_cold_trace_details for older trace data

Provide specific findings and actionable recommendations.`, serviceID, traceState, duration, toolInstructions)

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
	duration := args["duration"]

	if duration == "" {
		duration = defaultDuration
	}
	if logLevel == "" {
		logLevel = "ERROR"
	}

	prompt := fmt.Sprintf(`Analyze service logs using the query_logs tool:

**Tool Configuration:**
- query_logs with following parameters:
  - service_id: "%s" (if specified)
  - tags: [{"key": "level", "value": "%s"}] for log level filtering
  - duration: "%s" for time range
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

Provide specific log analysis findings and recommendations.`, serviceID, logLevel, duration)

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

	prompt := fmt.Sprintf(`Find top services using query_top_n_metrics tool:

**Tool Configuration:**
- query_top_n_metrics with parameters:
  - metrics_name: "%s"
  - top_n: %s
  - order: "%s" (DES for highest, ASC for lowest)
  - duration: "-1h" (or specify custom range)

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
- Use query_single_metrics for detailed service analysis
- Use query_traces for error investigation
- Use execute_mqe_expression for complex calculations

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

func traceDeepDiveHandler(_ context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := request.Params.Arguments
	traceID := args["trace_id"]
	view := args["view"]
	checkColdStorage := args["check_cold_storage"]

	if view == "" {
		view = "summary"
	}

	prompt := fmt.Sprintf(`Perform deep dive analysis of trace %s:

**Primary Analysis:**
- get_trace_details with trace_id: "%s" and view: "%s"
- Start with summary view for quick insights
- Use full view for complete span analysis
- Use errors_only view if trace has errors

**Cold Storage Check:**
- If trace not found in hot storage and check_cold_storage is "%s"
- Use get_cold_trace_details with same trace_id
- Check historical data for older traces

**Analysis Depth:**

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

Provide detailed trace analysis with specific optimization recommendations.`, traceID, traceID, view, checkColdStorage)

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
