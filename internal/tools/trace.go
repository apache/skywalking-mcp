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
	"errors"
	"fmt"
	"sort"

	"github.com/machinebox/graphql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	api "skywalking.apache.org/repo/goapi/query"

	"github.com/apache/skywalking-cli/pkg/graphql/client"
)

// AddTraceTools registers trace-related tools with the MCP server
func AddTraceTools(mcp *server.MCPServer) {
	TracesQueryTool.Register(mcp)
}

// View constants
const (
	ViewFull       = "full"
	ViewSummary    = "summary"
	ViewErrorsOnly = "errors_only"
)

// Query order constants
const (
	QueryOrderStartTime = "start_time"
	QueryOrderDuration  = "duration"
)

// Trace state constants
const (
	TraceStateSuccess = "success"
	TraceStateError   = "error"
	TraceStateAll     = "all"
)

// Error constants
const (
	ErrFailedToQueryTraces  = "failed to query traces: %v"
	ErrNoFilterCondition    = "at least one filter condition must be provided"
	ErrInvalidDurationRange = "invalid duration range: min_duration (%d) > max_duration (%d)"
	ErrNegativePageSize     = "page_size cannot be negative"
	ErrNegativePageNum      = "page_num cannot be negative"
	ErrInvalidTraceState    = "invalid trace_state '%s', available states: %s, %s, %s"
	ErrInvalidQueryOrder    = "invalid query_order '%s', available orders: %s, %s"
	ErrInvalidView          = "invalid view '%s', available views: %s, %s, %s"
	ErrNoTracesFound        = "no traces found matching the query criteria"
)

// queryTracesV2GQL is the GraphQL query for the trace-v2 protocol
const queryTracesV2GQL = `
query ($condition: TraceQueryCondition) {
	result: queryTraces(condition: $condition) {
		traces {
			spans {
				traceId segmentId spanId parentSpanId
				refs { traceId parentSegmentId parentSpanId type }
				serviceCode serviceInstanceName
				startTime endTime endpointName type peer component isError layer
				tags { key value }
				logs { time data { key value } }
				attachedEvents {
					startTime { seconds nanos } event endTime { seconds nanos }
					tags { key value } summary { key value }
				}
			}
		}
		retrievedTimeRange { startTime endTime }
	}
}`

// tracesV2 queries traces using the queryTraces (v2) protocol
func tracesV2(ctx context.Context, condition *api.TraceQueryCondition) (api.TraceList, error) {
	var response map[string]api.TraceList
	request := graphql.NewRequest(queryTracesV2GQL)
	request.Var("condition", condition)
	err := client.ExecuteQuery(ctx, request, &response)
	return response["result"], err
}

// Trace-specific constants
const (
	DefaultTracePageSize = 20
	DefaultTraceDuration = "1h"
)

// SpanTag represents a span tag for filtering traces
type SpanTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TracesQueryRequest defines the parameters for the traces query tool
type TracesQueryRequest struct {
	ServiceID          string    `json:"service_id,omitempty"`
	ServiceInstanceID  string    `json:"service_instance_id,omitempty"`
	TraceID            string    `json:"trace_id,omitempty"`
	EndpointID         string    `json:"endpoint_id,omitempty"`
	Start              string    `json:"start,omitempty"`
	End                string    `json:"end,omitempty"`
	Step               string    `json:"step,omitempty"`
	MinTraceDuration   int64     `json:"min_trace_duration,omitempty"`
	MaxTraceDuration   int64     `json:"max_trace_duration,omitempty"`
	TraceState         string    `json:"trace_state,omitempty"`
	QueryOrder         string    `json:"query_order,omitempty"`
	PageSize           int       `json:"page_size,omitempty"`
	PageNum            int       `json:"page_num,omitempty"`
	View               string    `json:"view,omitempty"`
	SlowTraceThreshold int64     `json:"slow_trace_threshold,omitempty"`
	Tags               []SpanTag `json:"tags,omitempty"`
	Cold               bool      `json:"cold,omitempty"`
}

// TracesSummary provides a high-level overview of multiple traces
type TracesSummary struct {
	TotalTraces  int                 `json:"total_traces"`
	SuccessCount int                 `json:"success_count"`
	ErrorCount   int                 `json:"error_count"`
	Services     []string            `json:"services"`
	Endpoints    []string            `json:"endpoints"`
	AvgDuration  float64             `json:"avg_duration_ms"`
	MinDuration  int64               `json:"min_duration_ms"`
	MaxDuration  int64               `json:"max_duration_ms"`
	TimeRange    TimeRange           `json:"time_range"`
	ErrorTraces  []BasicTraceSummary `json:"error_traces,omitempty"`
	SlowTraces   []BasicTraceSummary `json:"slow_traces,omitempty"`
}

// BasicTraceSummary provides essential information about a single trace
type BasicTraceSummary struct {
	TraceID      string `json:"trace_id"`
	ServiceName  string `json:"service_name"`
	EndpointName string `json:"endpoint_name"`
	StartTime    int64  `json:"start_time_ms"`
	Duration     int64  `json:"duration_ms"`
	IsError      bool   `json:"is_error"`
	SpanCount    int    `json:"span_count"`
}

// TimeRange represents the time span of the traces
type TimeRange struct {
	StartTime int64 `json:"start_time_ms"`
	EndTime   int64 `json:"end_time_ms"`
	Duration  int64 `json:"duration_ms"`
}

type spanStats struct {
	traceID   string
	rootSpan  *api.Span
	startTime int64
	endTime   int64
	isError   bool
}

func collectSpanStats(spans []*api.Span) spanStats {
	var s spanStats
	for _, span := range spans {
		if span == nil {
			continue
		}
		if span.SpanID == 0 && span.ParentSpanID == -1 && s.rootSpan == nil {
			s.rootSpan = span
		}
		if s.traceID == "" {
			s.traceID = span.TraceID
		}
		if s.startTime == 0 || span.StartTime < s.startTime {
			s.startTime = span.StartTime
		}
		if span.EndTime > s.endTime {
			s.endTime = span.EndTime
		}
		if span.IsError != nil && *span.IsError {
			s.isError = true
		}
	}
	return s
}

// createBasicTraceSummary creates a BasicTraceSummary from a TraceV2
func createBasicTraceSummary(traceItem *api.TraceV2) BasicTraceSummary {
	if traceItem == nil || len(traceItem.Spans) == 0 {
		return BasicTraceSummary{}
	}
	stats := collectSpanStats(traceItem.Spans)
	rootSpan := stats.rootSpan
	if rootSpan == nil {
		rootSpan = traceItem.Spans[0]
	}
	endpointName := ""
	if rootSpan.EndpointName != nil {
		endpointName = *rootSpan.EndpointName
	}
	return BasicTraceSummary{
		TraceID:      stats.traceID,
		ServiceName:  rootSpan.ServiceCode,
		EndpointName: endpointName,
		StartTime:    stats.startTime,
		Duration:     stats.endTime - stats.startTime,
		IsError:      stats.isError,
		SpanCount:    len(traceItem.Spans),
	}
}

// validateTracesQueryRequest validates traces query request parameters
func validateTracesQueryRequest(req *TracesQueryRequest) error {
	// At least one filter should be provided for meaningful results
	if req.ServiceID == "" && req.ServiceInstanceID == "" && req.TraceID == "" &&
		req.EndpointID == "" && req.Start == "" && req.End == "" && req.MinTraceDuration == 0 &&
		req.MaxTraceDuration == 0 {
		return errors.New(ErrNoFilterCondition)
	}

	// Validate duration range
	if req.MinTraceDuration > 0 && req.MaxTraceDuration > 0 && req.MinTraceDuration > req.MaxTraceDuration {
		return fmt.Errorf(ErrInvalidDurationRange, req.MinTraceDuration, req.MaxTraceDuration)
	}

	// Validate pagination
	if req.PageSize < 0 {
		return errors.New(ErrNegativePageSize)
	}
	if req.PageNum < 0 {
		return errors.New(ErrNegativePageNum)
	}

	return nil
}

// setBasicFields sets basic fields in the query condition
func setBasicFields(req *TracesQueryRequest, condition *api.TraceQueryCondition) {
	if req.ServiceID != "" {
		condition.ServiceID = &req.ServiceID
	}
	if req.ServiceInstanceID != "" {
		condition.ServiceInstanceID = &req.ServiceInstanceID
	}
	if req.TraceID != "" {
		condition.TraceID = &req.TraceID
	}
	if req.EndpointID != "" {
		condition.EndpointID = &req.EndpointID
	}

	if req.MinTraceDuration > 0 {
		minDuration := int(req.MinTraceDuration)
		condition.MinTraceDuration = &minDuration
	}
	if req.MaxTraceDuration > 0 {
		maxDuration := int(req.MaxTraceDuration)
		condition.MaxTraceDuration = &maxDuration
	}
}

// setTags sets tags in the query condition
func setTags(req *TracesQueryRequest, condition *api.TraceQueryCondition) {
	if len(req.Tags) > 0 {
		apiTags := make([]*api.SpanTag, len(req.Tags))
		for i, tag := range req.Tags {
			apiTags[i] = &api.SpanTag{
				Key:   tag.Key,
				Value: &tag.Value,
			}
		}
		condition.Tags = apiTags
	}
}

// setDuration sets duration in the query condition
func setDuration(req *TracesQueryRequest, condition *api.TraceQueryCondition, timeCtx TimeContext) {
	if req.Start != "" || req.End != "" {
		duration := BuildDurationWithContext(req.Start, req.End, req.Step, req.Cold, 60, timeCtx)
		condition.QueryDuration = &duration
		return
	}
	if req.TraceID == "" {
		// If no time range and no traceId provided, set default duration (last 1 hour)
		// SkyWalking OAP requires either queryDuration or traceId
		defaultDuration := ParseDurationWithContext(DefaultTraceDuration, req.Cold, timeCtx)
		condition.QueryDuration = &defaultDuration
	}
}

// setTraceState sets trace state in the query condition
func setTraceState(req *TracesQueryRequest, condition *api.TraceQueryCondition) error {
	switch req.TraceState {
	case TraceStateSuccess:
		condition.TraceState = api.TraceStateSuccess
	case TraceStateError:
		condition.TraceState = api.TraceStateError
	case TraceStateAll, "":
		condition.TraceState = api.TraceStateAll
	default:
		return fmt.Errorf(ErrInvalidTraceState,
			req.TraceState, TraceStateSuccess, TraceStateError, TraceStateAll)
	}
	return nil
}

// setQueryOrder sets query order in the query condition
func setQueryOrder(req *TracesQueryRequest, condition *api.TraceQueryCondition) error {
	switch req.QueryOrder {
	case QueryOrderStartTime, "":
		condition.QueryOrder = api.QueryOrderByStartTime
	case QueryOrderDuration:
		condition.QueryOrder = api.QueryOrderByDuration
	default:
		return fmt.Errorf(ErrInvalidQueryOrder,
			req.QueryOrder, QueryOrderStartTime, QueryOrderDuration)
	}
	return nil
}

// setPagination sets pagination in the query condition
func setPagination(req *TracesQueryRequest, condition *api.TraceQueryCondition) {
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = DefaultTracePageSize
	}
	condition.Paging = BuildPagination(req.PageNum, pageSize)
}

// buildQueryCondition builds the query condition from request parameters
func buildQueryCondition(req *TracesQueryRequest, timeCtx TimeContext) (*api.TraceQueryCondition, error) {
	condition := &api.TraceQueryCondition{
		TraceState: api.TraceStateAll,         // Default to all traces
		QueryOrder: api.QueryOrderByStartTime, // Default order
	}

	// Set basic fields
	setBasicFields(req, condition)

	// Set tags
	setTags(req, condition)

	// Set duration
	setDuration(req, condition, timeCtx)

	// Set trace state
	if err := setTraceState(req, condition); err != nil {
		return nil, err
	}

	// Set query order
	if err := setQueryOrder(req, condition); err != nil {
		return nil, err
	}

	// Set pagination
	setPagination(req, condition)

	return condition, nil
}

// searchTraces fetches traces based on query conditions
func searchTraces(ctx context.Context, req *TracesQueryRequest) (*mcp.CallToolResult, error) {
	if err := validateTracesQueryRequest(req); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Set default view
	if req.View == "" {
		req.View = ViewFull // Default to full view
	}

	// Build query condition
	timeCtx := GetTimeContext(ctx)
	condition, err := buildQueryCondition(req, timeCtx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Execute query using queryTraces (v2 protocol)
	traceList, err := tracesV2(ctx, condition)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrFailedToQueryTraces, err)), nil
	}

	return processTracesResult(&traceList, req.View, req.SlowTraceThreshold)
}

// processTracesResult handles the common logic for processing traces query results
func processTracesResult(traces *api.TraceList, view string, slowTraceThreshold int64) (*mcp.CallToolResult, error) {
	if traces == nil || len(traces.Traces) == 0 {
		return mcp.NewToolResultError(ErrNoTracesFound), nil
	}

	var result interface{}
	switch view {
	case ViewSummary:
		result = generateTracesSummary(traces, slowTraceThreshold)
	case ViewErrorsOnly:
		result = filterErrorTraces(traces)
	case ViewFull:
		result = traces
	default:
		return mcp.NewToolResultError(fmt.Sprintf(ErrInvalidView, view, ViewFull, ViewSummary, ViewErrorsOnly)), nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// processTraceItem processes a single TraceV2 item and updates summary statistics
func processTraceItem(traceItem *api.TraceV2, summary *TracesSummary,
	services, endpoints map[string]struct{}, durations *[]int64,
	errorTraces, slowTraces *[]BasicTraceSummary, slowTraceThreshold int64,
	minStartTime, maxEndTime *int64, totalDuration *int64) {
	if traceItem == nil || len(traceItem.Spans) == 0 {
		return
	}

	basic := createBasicTraceSummary(traceItem)
	if basic.TraceID == "" {
		return
	}

	endTimeMs := basic.StartTime + basic.Duration

	// Track time range
	if *minStartTime == 0 || basic.StartTime < *minStartTime {
		*minStartTime = basic.StartTime
	}
	if endTimeMs > *maxEndTime {
		*maxEndTime = endTimeMs
	}

	*durations = append(*durations, basic.Duration)
	*totalDuration += basic.Duration

	if basic.IsError {
		summary.ErrorCount++
		*errorTraces = append(*errorTraces, basic)
	} else {
		summary.SuccessCount++
	}

	if slowTraceThreshold > 0 && basic.Duration > slowTraceThreshold {
		*slowTraces = append(*slowTraces, basic)
	}

	// Collect services and endpoints from all spans
	for _, span := range traceItem.Spans {
		if span == nil {
			continue
		}
		if span.ServiceCode != "" {
			services[span.ServiceCode] = struct{}{}
		}
		if span.EndpointName != nil && *span.EndpointName != "" {
			endpoints[*span.EndpointName] = struct{}{}
		}
	}
}

// calculateStatistics calculates summary statistics from durations
func calculateStatistics(durations []int64, totalDuration int64) (avgDuration float64, minDuration, maxDuration int64) {
	if len(durations) == 0 {
		return 0, 0, 0
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	avgDuration = float64(totalDuration) / float64(len(durations))
	minDuration = durations[0]
	maxDuration = durations[len(durations)-1]

	return
}

// generateTracesSummary creates a comprehensive summary from multiple traces
func generateTracesSummary(traces *api.TraceList, slowTraceThreshold int64) *TracesSummary {
	if traces == nil || len(traces.Traces) == 0 {
		return &TracesSummary{}
	}

	summary := &TracesSummary{
		TotalTraces: len(traces.Traces),
	}

	services := make(map[string]struct{})
	endpoints := make(map[string]struct{})
	var durations []int64
	var errorTraces []BasicTraceSummary
	var slowTraces []BasicTraceSummary

	var minStartTime, maxEndTime int64
	var totalDuration int64

	// Process each trace item
	for _, traceItem := range traces.Traces {
		processTraceItem(traceItem, summary, services, endpoints, &durations,
			&errorTraces, &slowTraces, slowTraceThreshold, &minStartTime, &maxEndTime, &totalDuration)
	}

	// Calculate statistics
	summary.AvgDuration, summary.MinDuration, summary.MaxDuration =
		calculateStatistics(durations, totalDuration)

	// Set time range
	summary.TimeRange = TimeRange{
		StartTime: minStartTime,
		EndTime:   maxEndTime,
		Duration:  maxEndTime - minStartTime,
	}

	// Convert maps to slices
	for service := range services {
		summary.Services = append(summary.Services, service)
	}
	sort.Strings(summary.Services) // Ensure deterministic order
	for endpoint := range endpoints {
		summary.Endpoints = append(summary.Endpoints, endpoint)
	}

	// Sort error and slow traces by duration (descending)
	sort.Slice(errorTraces, func(i, j int) bool {
		return errorTraces[i].Duration > errorTraces[j].Duration
	})
	sort.Slice(slowTraces, func(i, j int) bool {
		return slowTraces[i].Duration > slowTraces[j].Duration
	})

	summary.ErrorTraces = errorTraces
	summary.SlowTraces = slowTraces

	return summary
}

// filterErrorTraces extracts only error traces from the results
func filterErrorTraces(traces *api.TraceList) []BasicTraceSummary {
	if traces == nil {
		return nil
	}

	var errorTraces []BasicTraceSummary
	for _, traceItem := range traces.Traces {
		if traceItem == nil {
			continue
		}
		basic := createBasicTraceSummary(traceItem)
		if basic.IsError {
			errorTraces = append(errorTraces, basic)
		}
	}

	// Sort by duration (descending) to show slowest errors first
	sort.Slice(errorTraces, func(i, j int) bool {
		return errorTraces[i].Duration > errorTraces[j].Duration
	})

	return errorTraces
}

// TracesQueryTool is a tool for querying traces with various conditions
var TracesQueryTool = NewTool(
	"query_traces",
	`This tool queries traces from SkyWalking OAP based on various conditions and provides intelligent data processing for LLM analysis.

Workflow:
1. Use this tool when you need to find traces matching specific criteria
2. Specify one or more query conditions to narrow down results
3. Use start/end to limit the time range for the search
4. Choose the appropriate view for your analysis needs

Query Conditions:
- service_id: Filter by specific service
- service_instance_id: Filter by specific service instance
- trace_id: Search for a specific trace ID
- endpoint_id: Filter by specific endpoint
- start/end: Time range for the query (e.g., start "-1h", end "now")
- min_trace_duration/max_trace_duration: Filter by trace duration in milliseconds
- trace_state: Filter by trace state (success, error, all)
- query_order: Sort order (start_time, duration, start_time_desc, duration_desc)
- view: Data presentation format (summary, errors_only, full)
- slow_trace_threshold: Optional threshold for identifying slow traces in milliseconds
- tags: Filter by span tags (key-value pairs)

Important Notes:
- SkyWalking OAP requires either a time range or 'trace_id' to be specified
- If no time range and no trace_id are provided, a default duration of "1h" (last 1 hour) will be used
- This ensures the query always has a valid time range or specific trace to search

View Options:
- 'full': (Default) Complete raw data for detailed analysis
- 'summary': Intelligent summary with performance metrics and insights
- 'errors_only': Focused list of error traces for troubleshooting

Best Practices:
- Start with 'summary' view to get an intelligent overview
- Use 'errors_only' view for focused troubleshooting
- Combine multiple filters for precise results
- Use time ranges to limit search scope and improve performance
- Only set slow_trace_threshold when you need to identify performance issues
- Use tags to filter traces by specific attributes or metadata

Examples:
- {"service_id": "Your_ApplicationName", "start": "-1h", "end": "now", "view": "summary"}: Recent traces summary with performance insights
- {"trace_state": "error", "start": "-7d", "end": "now", "view": "errors_only"}: Error traces from last week for troubleshooting
- {"min_trace_duration": 1000, "query_order": "duration_desc", "view": "summary"}: Slow traces analysis with performance metrics
- {"slow_trace_threshold": 5000, "view": "summary"}: Identify traces slower than 5 seconds
- {"service_id": "Your_ApplicationName"}: Query with default 1-hour duration
- {"tags": [{"key": "http.method", "value": "POST"}, {"key": "http.status_code", "value": "500"}], 
	"start": "-1h", "end": "now"}: Find traces with specific HTTP tags`,
	searchTraces,
	mcp.WithTitleAnnotation("Query traces with intelligent analysis"),
	mcp.WithString("service_id",
		mcp.Description("Service ID to filter traces. Use this to find traces from a specific service."),
	),
	mcp.WithString("service_instance_id",
		mcp.Description("Service instance ID to filter traces. Use this to find traces from a specific instance."),
	),
	mcp.WithString("trace_id",
		mcp.Description("Specific trace ID to search for. Use this when you know the exact trace ID."),
	),
	mcp.WithString("endpoint_id",
		mcp.Description("Endpoint ID to filter traces. Use this to find traces for a specific endpoint."),
	),
	mcp.WithString("start",
		mcp.Description("Start time for the query. Examples: \"2023-01-01 12:00:00\", \"-1h\" (1 hour ago), \"-30m\" (30 minutes ago)"),
	),
	mcp.WithString("end",
		mcp.Description("End time for the query. Examples: \"2023-01-01 13:00:00\", \"now\","+
			" \"-10m\" (10 minutes ago) Defaults to current time if omitted."),
	),
	mcp.WithString("step",
		mcp.Enum("SECOND", "MINUTE", "HOUR", "DAY"),
		mcp.Description("Time step granularity. If not specified, uses adaptive sizing."),
	),
	mcp.WithNumber("min_trace_duration",
		mcp.Description("Minimum trace duration in milliseconds. Use this to filter out fast traces."),
	),
	mcp.WithNumber("max_trace_duration",
		mcp.Description("Maximum trace duration in milliseconds. Use this to filter out slow traces."),
	),
	mcp.WithString("trace_state",
		mcp.Enum(TraceStateSuccess, TraceStateError, TraceStateAll),
		mcp.Description(`Filter traces by their state:
- 'success': Only successful traces
- 'error': Only traces with errors
- 'all': All traces (default)`),
	),
	mcp.WithString("query_order",
		mcp.Enum(QueryOrderStartTime, QueryOrderDuration),
		mcp.Description(`Sort order for results:
- 'start_time': Oldest first
- 'duration': Shortest first`),
	),
	mcp.WithString("view",
		mcp.Enum(ViewSummary, ViewErrorsOnly, ViewFull),
		mcp.Description(`Data presentation format:
- 'full': (Default) Complete raw data for detailed analysis
- 'summary': Intelligent summary with performance metrics and insights
- 'errors_only': Focused list of error traces for troubleshooting`),
	),
	mcp.WithNumber("slow_trace_threshold",
		mcp.Description("Optional threshold for identifying slow traces in milliseconds. "+
			"Only when this parameter is set will slow traces be included in the summary. "+
			"Traces with duration exceeding this threshold will be listed in slow_traces. "+
			"Examples: 500 (0.5s), 2000 (2s), 5000 (5s)"),
	),
	mcp.WithArray("tags",
		mcp.Description(`Array of span tags to filter traces. Each tag should have 'key' and 'value' fields.
Examples: [{"key": "http.method", "value": "POST"}, {"key": "http.status_code", "value": "500"}]`),
		mcp.Items(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":   map[string]any{"type": "string"},
				"value": map[string]any{"type": "string"},
			},
			"required": []string{"key", "value"},
		}),
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)
