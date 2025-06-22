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
	"time"

	api "skywalking.apache.org/repo/goapi/query"
)

// Default values
const (
	DefaultPageSize = 15
	DefaultPageNum  = 1
	DefaultStep     = "MINUTE"
	DefaultDuration = 30 // minutes
)

// Error messages
const (
	ErrMissingDuration = "missing required parameter: duration"
	ErrMarshalFailed   = "failed to marshal result: %v"
)

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
		step = determineStep(duration)
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
		if step == "" || !stepEnum.IsValid() {
			stepEnum = DefaultStep
		}
		return api.Duration{
			Start:     start,
			End:       end,
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

// determineStep determines the step based on the duration
func determineStep(duration time.Duration) api.Step {
	if duration >= 24*time.Hour {
		return api.StepDay
	} else if duration >= time.Hour {
		return api.StepHour
	} else if duration >= time.Minute {
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
