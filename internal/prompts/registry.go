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
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

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
			{Name: "start", Description: `Start of the analysis window. Examples: "-1h", "-30m", "2024-01-01 12:00:00". Default: -1h`, Required: false},
			{Name: "end", Description: `End of the analysis window. Examples: "now", "2024-01-01 13:00:00". Default: now`, Required: false},
		},
	}, performanceAnalysisHandler)

	// Service Comparison Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "compare-services",
		Description: "Compare performance metrics between multiple services",
		Arguments: []mcp.PromptArgument{
			{Name: "services", Description: "Comma-separated list of service names to compare", Required: true},
			{Name: "metrics", Description: "Metrics to compare (response_time, sla, cpm, all)", Required: false},
			{Name: "start", Description: `Start of the comparison window. Examples: "-1h", "-2h", "-1d". Default: -1h`, Required: false},
			{Name: "end", Description: `End of the comparison window. Examples: "now", "2024-01-01 13:00:00". Default: now`, Required: false},
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
			{Name: "start", Description: `Start of the search window. Examples: "-1h" (last hour), "-30m" (last 30 minutes). Default: -1h`, Required: false},
			{Name: "end", Description: `End of the search window. Examples: "now", "2024-01-01 13:00:00". Default: now`, Required: false},
		},
	}, traceInvestigationHandler)

	// Trace Deep Dive
	s.AddPrompt(mcp.Prompt{
		Name:        "trace-deep-dive",
		Description: "Deep dive analysis of a specific trace",
		Arguments: []mcp.PromptArgument{
			{Name: "trace_id", Description: "The trace ID to analyze", Required: true},
			{Name: "view", Description: "Analysis view (full, summary, errors_only)", Required: false},
		},
	}, traceDeepDiveHandler)

	// Log Analysis Prompt
	s.AddPrompt(mcp.Prompt{
		Name:        "analyze-logs",
		Description: "Analyze service logs for errors and patterns",
		Arguments: []mcp.PromptArgument{
			{Name: "service_id", Description: "Service to analyze logs", Required: false},
			{Name: "log_level", Description: "Log level to filter (ERROR, WARN, INFO)", Required: false},
			{Name: "start", Description: `Start of the analysis window. Examples: "-1h" (last hour), "-6h" (last 6 hours). Default: -1h`, Required: false},
			{Name: "end", Description: `End of the analysis window. Examples: "now", "2024-01-01 13:00:00". Default: now`, Required: false},
		},
	}, logAnalysisHandler)
}

func addUtilityPrompts(s *server.MCPServer) {
	// Service Topology Explorer
	s.AddPrompt(mcp.Prompt{
		Name:        "explore-service-topology",
		Description: "Explore the service topology of a layer: list services, instances, endpoints, and processes within a time range",
		Arguments: []mcp.PromptArgument{
			{Name: "layer", Description: "The layer to explore (e.g. GENERAL, MESH, K8S). Use list_layers if unknown.", Required: true},
			{Name: "start", Description: `Start time for the query. Examples: "2024-01-01 12:00:00", "-1h" (1 hour ago).`, Required: true},
			{Name: "end", Description: `End time for the query. Examples: "2024-01-01 13:00:00", "now".` +
				` Defaults to current time if omitted.`, Required: false},
		},
	}, exploreServiceTopologyHandler)

	// Generate Duration Prompt
	s.AddPrompt(mcp.Prompt{
		Name: "generate_duration",
		Description: "Convert a natural-language time range into a {start, end} duration object" +
			" for use with list_instances, list_endpoints, list_processes, and similar tools",
		Arguments: []mcp.PromptArgument{
			{Name: "time_range", Description: `Natural-language description of the desired time range.` +
				` Examples: "last hour", "past 30 minutes", "yesterday 9am to 5pm", "2024-01-01 12:00 to 13:00"`,
				Required: true},
		},
	}, generateDurationHandler)

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
