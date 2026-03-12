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

import "fmt"

// Tool capability mapping for different analysis types
var toolCapabilities = map[string][]string{
	"performance_analysis": {
		"execute_mqe_expression",
		"query_traces",
	},
	"trace_investigation": {
		"query_traces",
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
		"execute_mqe_expression",
	},
	"metrics_exploration": {
		"list_mqe_metrics",
		"get_mqe_metric_type",
	},
}

// Analysis execution chains for different types of analysis
var analysisChains = map[string][]struct {
	Tool    string
	Purpose string
}{
	"performance_analysis": {
		{Tool: "execute_mqe_expression", Purpose: "Query metrics like CPM, SLA, response time, percentiles, and top entities"},
		{Tool: "query_traces", Purpose: "Find error traces for deeper investigation"},
	},
	"trace_investigation": {
		{Tool: "query_traces", Purpose: "Search for traces with specific filters and analyze results"},
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
