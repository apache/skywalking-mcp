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
	"fmt"
	"strings"
	"time"

	api "skywalking.apache.org/repo/goapi/query"
)

// Default values
const (
	DefaultPageSize = 15
	DefaultPageNum  = 1
	DefaultDuration = 30 // minutes
	nowKeyword      = "now"
)

// Error messages
const (
	ErrMissingDuration = "missing required parameter: duration"
	ErrMarshalFailed   = "failed to marshal result: %v"
)

// FinalizeURL ensures the URL ends with "/graphql".
func FinalizeURL(urlStr string) string {
	if !strings.HasSuffix(urlStr, "/graphql") {
		urlStr = strings.TrimRight(urlStr, "/") + "/graphql"
	}
	return urlStr
}

// FormatTimeByStep formats time according to step granularity
func FormatTimeByStep(t time.Time, step api.Step) string {
	switch step {
	case api.StepDay:
		return t.Format("2006-01-02")
	case api.StepHour:
		return t.Format("2006-01-02 15")
	case api.StepMinute:
		return t.Format("2006-01-02 1504")
	case api.StepSecond:
		return t.Format("2006-01-02 150405")
	default:
		return t.Format("2006-01-02 15:04:05")
	}
}

// ParseDuration converts duration string to api.Duration
func ParseDuration(durationStr string, coldStage bool) api.Duration {
	now := time.Now()
	var startTime, endTime time.Time
	var step api.Step

	duration, err := time.ParseDuration(durationStr)
	if err == nil {
		if duration < 0 {
			startTime = now.Add(duration)
			endTime = now
		} else {
			startTime = now
			endTime = now.Add(duration)
		}
		// Use adaptive step based on time range
		step = determineAdaptiveStep(startTime, endTime)
	} else {
		startTime, endTime, step = parseLegacyDuration(durationStr)
	}

	if !step.IsValid() {
		step = api.StepMinute
	}

	return api.Duration{
		Start:     FormatTimeByStep(startTime, step),
		End:       FormatTimeByStep(endTime, step),
		Step:      step,
		ColdStage: &coldStage,
	}
}

// BuildPagination creates pagination with defaults
func BuildPagination(pageNum, pageSize int) *api.Pagination {
	if pageNum <= 0 {
		pageNum = DefaultPageNum
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	return &api.Pagination{
		PageNum:  &pageNum,
		PageSize: pageSize,
	}
}

// BuildDuration creates duration from parameters
func BuildDuration(start, end, step string, cold bool, defaultDurationMinutes int) api.Duration {
	if start != "" || end != "" {
		stepEnum := api.Step(step)
		// Parse and format start and end times
		startTime, endTime := parseStartEndTimes(start, end)

		// If step is not provided or invalid, determine it adaptively based on time range
		if step == "" || !stepEnum.IsValid() {
			stepEnum = determineAdaptiveStep(startTime, endTime)
		}

		return api.Duration{
			Start:     FormatTimeByStep(startTime, stepEnum),
			End:       FormatTimeByStep(endTime, stepEnum),
			Step:      stepEnum,
			ColdStage: &cold,
		}
	}

	if defaultDurationMinutes <= 0 {
		defaultDurationMinutes = DefaultDuration
	}
	defaultDurationStr := fmt.Sprintf("-%dm", defaultDurationMinutes)
	return ParseDuration(defaultDurationStr, cold)
}

// determineAdaptiveStep determines the adaptive step based on the time range
func determineAdaptiveStep(startTime, endTime time.Time) api.Step {
	duration := endTime.Sub(startTime)
	if duration >= 7*24*time.Hour {
		return api.StepDay
	} else if duration >= 24*time.Hour {
		return api.StepHour
	} else if duration >= time.Hour {
		return api.StepMinute
	}

	return api.StepSecond
}

// parseLegacyDuration parses legacy duration strings like "7d", "24h"
func parseLegacyDuration(durationStr string) (startTime, endTime time.Time, step api.Step) {
	now := time.Now()
	if len(durationStr) > 1 && (durationStr[len(durationStr)-1] == 'd' || durationStr[len(durationStr)-1] == 'D') {
		var days int
		if _, parseErr := fmt.Sscanf(durationStr[:len(durationStr)-1], "%d", &days); parseErr == nil && days > 0 {
			startTime = now.AddDate(0, 0, -days)
			endTime = now
			step = api.StepDay
			return startTime, endTime, step
		}
		startTime = now.AddDate(0, 0, -7)
		endTime = now
		step = api.StepDay
		return startTime, endTime, step
	}
	if len(durationStr) > 1 && (durationStr[len(durationStr)-1] == 'h' || durationStr[len(durationStr)-1] == 'H') {
		var hours int
		if _, parseErr := fmt.Sscanf(durationStr[:len(durationStr)-1], "%d", &hours); parseErr == nil && hours > 0 {
			startTime = now.Add(-time.Duration(hours) * time.Hour)
			endTime = now
			step = api.StepHour
			return startTime, endTime, step
		}
		startTime = now.Add(-1 * time.Hour)
		endTime = now
		step = api.StepHour
		return startTime, endTime, step
	}
	startTime = now.AddDate(0, 0, -7)
	endTime = now
	step = api.StepDay
	return startTime, endTime, step
}

// parseAbsoluteTime tries to parse absolute time in various formats
func parseAbsoluteTime(timeStr string) (time.Time, bool) {
	timeFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 1504",
		"2006-01-02 15",
		"2006-01-02 150405",
		"2006-01-02",
	}

	for _, format := range timeFormats {
		if parsed, err := time.Parse(format, timeStr); err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}

// parseTimeString parses a time string (start or end)
func parseTimeString(timeStr string, defaultTime time.Time) time.Time {
	now := time.Now()

	if timeStr == "" {
		return defaultTime
	}

	if strings.EqualFold(timeStr, nowKeyword) {
		return now
	}

	// Try relative time like "-30m", "-1h"
	if duration, err := time.ParseDuration(timeStr); err == nil {
		return now.Add(duration)
	}

	// Try absolute time
	if parsed, ok := parseAbsoluteTime(timeStr); ok {
		return parsed
	}

	return defaultTime
}

// parseStartEndTimes parses start and end time strings
func parseStartEndTimes(start, end string) (startTime, endTime time.Time) {
	now := time.Now()
	defaultStart := now.Add(-30 * time.Minute) // Default to 30 minutes ago

	startTime = parseTimeString(start, defaultStart)
	endTime = parseTimeString(end, now)

	return startTime, endTime
}
