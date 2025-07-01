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
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	api "skywalking.apache.org/repo/goapi/query"

	"github.com/apache/skywalking-cli/pkg/graphql/trace"
)

// AddTraceTools registers trace-related tools with the MCP server
func AddTraceTools(mcp *server.MCPServer) {
	SearchTraceTool.Register(mcp)
	ColdTraceTool.Register(mcp)
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
	ErrMissingTraceID         = "missing required parameter: trace_id"
	ErrFailedToQueryTrace     = "failed to query trace '%s': %v"
	ErrFailedToQueryColdTrace = "failed to query cold trace '%s': %v"
	ErrFailedToQueryTraces    = "failed to query traces: %v"
	ErrNoFilterCondition      = "at least one filter condition must be provided"
	ErrInvalidDurationRange   = "invalid duration range: min_duration (%d) > max_duration (%d)"
	ErrNegativePageSize       = "page_size cannot be negative"
	ErrNegativePageNum        = "page_num cannot be negative"
	ErrInvalidTraceState      = "invalid trace_state '%s', available states: %s, %s, %s"
	ErrInvalidQueryOrder      = "invalid query_order '%s', available orders: %s, %s"
	ErrTraceNotFound          = "trace with ID '%s' not found"
	ErrInvalidView            = "invalid view '%s', available views: %s, %s, %s"
	ErrNoTracesFound          = "no traces found matching the query criteria"
)

const TimeFormatFull = "2006-01-02 15:04:05"

// Trace-specific constants
const (
	DefaultTracePageSize = 20
	DefaultTraceDuration = "1h"
)

// TraceRequest defines the parameters for the trace tool
type TraceRequest struct {
	TraceID string `json:"trace_id"`
	View    string `json:"view,omitempty"`
}

// ColdTraceRequest defines the parameters for the cold trace tool
type ColdTraceRequest struct {
	TraceID  string `json:"trace_id"`
	Duration string `json:"duration"`
	View     string `json:"view,omitempty"`
}

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
	Duration           string    `json:"duration,omitempty"`
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

// TraceSummary provides a high-level overview of a trace
type TraceSummary struct {
	TraceID       string   `json:"trace_id"`
	TotalSpans    int      `json:"total_spans"`
	Services      []string `json:"services"`
	TotalDuration int64    `json:"total_duration_ms"`
	ErrorCount    int      `json:"error_count"`
	HasErrors     bool     `json:"has_errors"`
	RootEndpoint  string   `json:"root_endpoint"`
	StartTime     int64    `json:"start_time_ms"`
	EndTime       int64    `json:"end_time_ms"`
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

// createBasicTraceSummary creates a BasicTraceSummary from trace item data
func createBasicTraceSummary(traceItem *api.BasicTrace, startTimeMs, duration int64, isError bool) BasicTraceSummary {
	return BasicTraceSummary{
		TraceID:      traceItem.TraceIds[0], // Use first trace ID
		ServiceName:  traceItem.SegmentID,   // Use segment ID as service name
		EndpointName: strings.Join(traceItem.EndpointNames, ", "),
		StartTime:    startTimeMs,
		Duration:     duration,
		IsError:      isError,
		SpanCount:    0, // BasicTrace doesn't have span count
	}
}

// processTraceResult handles the common logic for processing trace results
func processTraceResult(traceID string, traceData *api.Trace, view string) (*mcp.CallToolResult, error) {
	if len(traceData.Spans) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf(ErrTraceNotFound, traceID)), nil
	}

	var result interface{}
	switch view {
	case ViewSummary:
		result = generateTraceSummary(traceID, traceData)
	case ViewErrorsOnly:
		result = filterErrorSpans(traceData)
	case ViewFull:
		result = traceData
	default:
		return mcp.NewToolResultError(fmt.Sprintf(ErrInvalidView, view, ViewFull, ViewSummary, ViewErrorsOnly)), nil
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrMarshalFailed, err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// validateTraceRequest validates trace request parameters
func validateTraceRequest(req TraceRequest) error {
	if req.TraceID == "" {
		return errors.New(ErrMissingTraceID)
	}
	return nil
}

// validateColdTraceRequest validates cold trace request parameters
func validateColdTraceRequest(req ColdTraceRequest) error {
	if req.TraceID == "" {
		return errors.New(ErrMissingTraceID)
	}
	if req.Duration == "" {
		return errors.New(ErrMissingDuration)
	}
	return nil
}

// searchTrace fetches the trace data and processes it based on the requested view
func searchTrace(ctx context.Context, req *TraceRequest) (*mcp.CallToolResult, error) {
	if err := validateTraceRequest(*req); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if req.View == "" {
		req.View = ViewFull // Set default value
	}

	traces, err := trace.Trace(ctx, req.TraceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrFailedToQueryTrace, req.TraceID, err)), nil
	}
	traceData := &traces

	return processTraceResult(req.TraceID, traceData, req.View)
}

// searchColdTrace fetches the trace data from cold storage and processes it based on the requested view
func searchColdTrace(ctx context.Context, req *ColdTraceRequest) (*mcp.CallToolResult, error) {
	if err := validateColdTraceRequest(*req); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if req.View == "" {
		req.View = ViewFull // Set default value
	}

	// Parse duration string to api.Duration
	duration := ParseDuration(req.Duration, true)

	traces, err := trace.ColdTrace(ctx, duration, req.TraceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrFailedToQueryColdTrace, req.TraceID, err)), nil
	}
	traceData := &traces

	return processTraceResult(req.TraceID, traceData, req.View)
}

// generateTraceSummary creates a summary view from full trace data
func generateTraceSummary(traceID string, traceData *api.Trace) *TraceSummary {
	summary := &TraceSummary{
		TraceID:    traceID,
		TotalSpans: len(traceData.Spans),
	}
	services := make(map[string]struct{})

	for _, span := range traceData.Spans {
		if span == nil {
			continue
		}
		services[span.ServiceCode] = struct{}{}
		if span.IsError != nil && *span.IsError {
			summary.ErrorCount++
		}
		// Heuristic to find the root span: a span with spanId 0 and parentSpanId -1
		if span.SpanID == 0 && span.ParentSpanID == -1 {
			if span.EndpointName != nil {
				summary.RootEndpoint = *span.EndpointName
			}
			summary.StartTime = span.StartTime
			summary.EndTime = span.EndTime
			if summary.StartTime > 0 && summary.EndTime > 0 {
				summary.TotalDuration = summary.EndTime - summary.StartTime
			}
		}
	}

	summary.HasErrors = summary.ErrorCount > 0
	for service := range services {
		summary.Services = append(summary.Services, service)
	}
	sort.Strings(summary.Services) // Ensure deterministic order
	return summary
}

// filterErrorSpans extracts only the spans with errors from full trace data
func filterErrorSpans(traceData *api.Trace) []*api.Span {
	var errorSpans []*api.Span
	for _, span := range traceData.Spans {
		if span != nil && span.IsError != nil && *span.IsError {
			errorSpans = append(errorSpans, span)
		}
	}
	return errorSpans
}

// validateTracesQueryRequest validates traces query request parameters
func validateTracesQueryRequest(req *TracesQueryRequest) error {
	// At least one filter should be provided for meaningful results
	if req.ServiceID == "" && req.ServiceInstanceID == "" && req.TraceID == "" &&
		req.EndpointID == "" && req.Duration == "" && req.MinTraceDuration == 0 &&
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
func setDuration(req *TracesQueryRequest, condition *api.TraceQueryCondition) {
	if req.Duration != "" {
		duration := ParseDuration(req.Duration, req.Cold)
		condition.QueryDuration = &duration
	} else if req.TraceID == "" {
		// If no duration and no traceId provided, set default duration (last 1 hour)
		// SkyWalking OAP requires either queryDuration or traceId
		defaultDuration := ParseDuration(DefaultTraceDuration, req.Cold)
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
func buildQueryCondition(req *TracesQueryRequest) (*api.TraceQueryCondition, error) {
	condition := &api.TraceQueryCondition{
		TraceState: api.TraceStateAll,         // Default to all traces
		QueryOrder: api.QueryOrderByStartTime, // Default order
	}

	// Set basic fields
	setBasicFields(req, condition)

	// Set tags
	setTags(req, condition)

	// Set duration
	setDuration(req, condition)

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
	condition, err := buildQueryCondition(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Execute query
	traces, err := trace.Traces(ctx, condition)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(ErrFailedToQueryTraces, err)), nil
	}

	return processTracesResult(&traces, req.View, req.SlowTraceThreshold)
}

// processTracesResult handles the common logic for processing traces query results
func processTracesResult(traces *api.TraceBrief, view string, slowTraceThreshold int64) (*mcp.CallToolResult, error) {
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

// processTraceItem processes a single trace item and updates summary statistics
func processTraceItem(traceItem *api.BasicTrace, summary *TracesSummary,
	services, endpoints map[string]struct{}, durations *[]int64,
	errorTraces, slowTraces *[]BasicTraceSummary, slowTraceThreshold int64,
	minStartTime, maxEndTime *int64, totalDuration *int64) {
	if traceItem == nil {
		return
	}

	// Parse start time
	startTime, err := time.Parse(TimeFormatFull, traceItem.Start)
	if err != nil {
		return // Skip invalid traces
	}
	startTimeMs := startTime.UnixMilli()
	endTimeMs := startTimeMs + int64(traceItem.Duration)

	// Track time range
	if *minStartTime == 0 || startTimeMs < *minStartTime {
		*minStartTime = startTimeMs
	}
	if endTimeMs > *maxEndTime {
		*maxEndTime = endTimeMs
	}

	// Calculate duration
	duration := int64(traceItem.Duration)
	*durations = append(*durations, duration)
	*totalDuration += duration

	// Count errors
	isError := traceItem.IsError != nil && *traceItem.IsError
	if isError {
		summary.ErrorCount++
		*errorTraces = append(*errorTraces, createBasicTraceSummary(traceItem, startTimeMs, duration, true))
	} else {
		summary.SuccessCount++
	}

	// Identify slow traces only if threshold is configured
	if slowTraceThreshold > 0 && duration > slowTraceThreshold {
		*slowTraces = append(*slowTraces, createBasicTraceSummary(traceItem, startTimeMs, duration, isError))
	}

	// Collect services and endpoints
	services[traceItem.SegmentID] = struct{}{}
	for _, endpoint := range traceItem.EndpointNames {
		if endpoint != "" {
			endpoints[endpoint] = struct{}{}
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
func generateTracesSummary(traces *api.TraceBrief, slowTraceThreshold int64) *TracesSummary {
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
func filterErrorTraces(traces *api.TraceBrief) []BasicTraceSummary {
	if traces == nil {
		return nil
	}

	var errorTraces []BasicTraceSummary
	for _, traceItem := range traces.Traces {
		if traceItem != nil && traceItem.IsError != nil && *traceItem.IsError {
			// Parse start time
			startTime, err := time.Parse(TimeFormatFull, traceItem.Start)
			if err != nil {
				continue
			}
			startTimeMs := startTime.UnixMilli()

			errorTraces = append(errorTraces,
				createBasicTraceSummary(traceItem, startTimeMs, int64(traceItem.Duration), true))
		}
	}

	// Sort by duration (descending) to show slowest errors first
	sort.Slice(errorTraces, func(i, j int) bool {
		return errorTraces[i].Duration > errorTraces[j].Duration
	})

	return errorTraces
}

// SearchTraceTool is a tool for searching traces by trace ID with different views
var SearchTraceTool = NewTool[TraceRequest, *mcp.CallToolResult](
	"get_trace_details",
	`This tool provides detailed information about a distributed trace from SkyWalking OAP.

Workflow:
1. Use this tool when you need to analyze a specific trace by its trace ID
2. Choose the appropriate view based on your analysis needs:
   - 'full': For complete trace analysis with all spans and details
   - 'summary': For quick overview and performance metrics
   - 'errors_only': For troubleshooting and error investigation

Best Practices:
- Use 'summary' view first to get an overview of the trace
- Switch to 'errors_only' if the summary shows errors
- Use 'full' view for detailed debugging and span-by-span analysis
- Trace IDs are typically found in logs, error messages, or monitoring dashboards

Examples:
- {"trace_id": "abc123..."}: Get complete trace details for analysis
- {"trace_id": "abc123...", "view": "summary"}: Quick performance overview
- {"trace_id": "abc123...", "view": "errors_only"}: Focus on error spans only`,
	searchTrace,
	mcp.WithTitleAnnotation("Search a trace by TraceId"),
	mcp.WithString("trace_id", mcp.Required(),
		mcp.Description(`The unique identifier of the trace to retrieve.`),
	),
	mcp.WithString("view",
		mcp.Enum(ViewFull, ViewSummary, ViewErrorsOnly),
		mcp.Description(`Specifies the level of detail for trace analysis:
- 'full': (Default) Complete trace with all spans, service calls, and metadata
- 'summary': High-level overview with services, duration, and error count
- 'errors_only': Only spans marked as errors for troubleshooting`),
	),
)

// ColdTraceTool is a tool for searching traces from cold storage by trace ID with different views
var ColdTraceTool = NewTool[ColdTraceRequest, *mcp.CallToolResult](
	"get_cold_trace_details",
	`This tool queries BanyanDB cold storage for historical trace data that may no longer be available in hot storage.

Important Notes:
- Only works with BanyanDB storage backend
- Queries older trace data that has been moved to cold storage
- May have slower response times compared to hot storage queries
- Use when trace data is not found in regular trace queries

Duration Format:
- Standard Go duration: "7d", "1h", "-30m", "2h30m"
- Negative values mean "ago": "-7d" = 7 days ago to now
- Positive values mean "from now": "2h" = now to 2 hours later
- Legacy format: "6d", "12h" (backward compatible)

Usage Scenarios:
- Historical incident investigation
- Long-term performance analysis
- Compliance and audit requirements
- When hot storage queries return no results

Examples:
- {"trace_id": "abc123...", "duration": "7d"}: Search last 7 days of cold storage
- {"trace_id": "abc123...", "duration": "-30m"}: Search from 30 minutes ago to now
- {"trace_id": "abc123...", "duration": "1h", "view": "summary"}: Quick summary from last hour
- {"trace_id": "abc123...", "duration": "2h30m", "view": "errors_only"}: Error analysis from last 2.5 hours`,
	searchColdTrace,
	mcp.WithTitleAnnotation("Search a cold trace by TraceId"),
	mcp.WithString("trace_id", mcp.Required(),
		mcp.Description(`The unique identifier of the trace to retrieve from cold storage. Use this when regular trace queries return no results.`),
	),
	mcp.WithString("duration", mcp.Required(),
		mcp.Description(`Time duration for cold storage query. Examples: "7d" (last 7 days), "-30m" (last 30 minutes), "2h30m" (last 2.5 hours)`),
	),
	mcp.WithString("view",
		mcp.Enum(ViewFull, ViewSummary, ViewErrorsOnly),
		mcp.Description(`Specifies the level of detail for cold trace analysis:
- 'full': (Default) Complete trace with all spans from cold storage
- 'summary': High-level overview with services, duration, and error count
- 'errors_only': Only error spans for focused troubleshooting`),
	),
)

// TracesQueryTool is a tool for querying traces with various conditions
var TracesQueryTool = NewTool[TracesQueryRequest, *mcp.CallToolResult](
	"query_traces",
	`This tool queries traces from SkyWalking OAP based on various conditions and provides intelligent data processing for LLM analysis.

Workflow:
1. Use this tool when you need to find traces matching specific criteria
2. Specify one or more query conditions to narrow down results
3. Use duration to limit the time range for the search
4. Choose the appropriate view for your analysis needs

Query Conditions:
- service_id: Filter by specific service
- service_instance_id: Filter by specific service instance
- trace_id: Search for a specific trace ID
- endpoint_id: Filter by specific endpoint
- duration: Time range for the query (e.g., "1h", "7d", "-30m")
- min_trace_duration/max_trace_duration: Filter by trace duration in milliseconds
- trace_state: Filter by trace state (success, error, all)
- query_order: Sort order (start_time, duration, start_time_desc, duration_desc)
- view: Data presentation format (summary, errors_only, full)
- slow_trace_threshold: Optional threshold for identifying slow traces in milliseconds
- tags: Filter by span tags (key-value pairs)

Important Notes:
- SkyWalking OAP requires either 'duration' or 'trace_id' to be specified
- If neither is provided, a default duration of "1h" (last 1 hour) will be used
- This ensures the query always has a valid time range or specific trace to search

View Options:
- 'full': (Default) Complete raw data for detailed analysis
- 'summary': Intelligent summary with performance metrics and insights
- 'errors_only': Focused list of error traces for troubleshooting

Best Practices:
- Start with 'summary' view to get an intelligent overview
- Use 'errors_only' view for focused troubleshooting
- Combine multiple filters for precise results
- Use duration to limit search scope and improve performance
- Only set slow_trace_threshold when you need to identify performance issues
- Use tags to filter traces by specific attributes or metadata

Examples:
- {"service_id": "Your_ApplicationName", "duration": "1h", "view": "summary"}: Recent traces summary with performance insights
- {"trace_state": "error", "duration": "7d", "view": "errors_only"}: Error traces from last week for troubleshooting
- {"min_trace_duration": 1000, "query_order": "duration_desc", "view": "summary"}: Slow traces analysis with performance metrics
- {"slow_trace_threshold": 5000, "view": "summary"}: Identify traces slower than 5 seconds
- {"service_id": "Your_ApplicationName"}: Query with default 1-hour duration
- {"tags": [{"key": "http.method", "value": "POST"}, {"key": "http.status_code", "value": "500"}], 
  "duration": "1h"}: Find traces with specific HTTP tags`,
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
	mcp.WithString("duration",
		mcp.Description(`Time duration for the query. Examples: "7d" (last 7 days), "-30m" (last 30 minutes), "2h30m" (last 2.5 hours)`),
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
	),
	mcp.WithBoolean("cold",
		mcp.Description("Whether to query from cold-stage storage. Set to true for historical data queries."),
	),
)
