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
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	api "skywalking.apache.org/repo/goapi/query"
)

// The jsonschema struct tag carries only descriptions, so enums are applied to
// the inferred schema afterwards. Dropping that step fails silently.
func TestAlarmQueryToolSchema(t *testing.T) {
	schema, ok := alarmQueryTool().InputSchema.(*jsonschema.Schema)
	if !ok {
		t.Fatalf("input schema type = %T", alarmQueryTool().InputSchema)
	}

	tests := []struct {
		property  string
		enumCount int
	}{
		{property: "scope", enumCount: 9},
		{property: "step", enumCount: 4},
		{property: "keyword", enumCount: 0},
		{property: "page_num", enumCount: 0},
	}

	for _, tc := range tests {
		t.Run(tc.property, func(t *testing.T) {
			property, ok := schema.Properties[tc.property]
			if !ok {
				t.Fatalf("property %q missing", tc.property)
			}
			if len(property.Enum) != tc.enumCount {
				t.Fatalf("property %q enum count = %d, want %d", tc.property, len(property.Enum), tc.enumCount)
			}
			if property.Description == "" {
				t.Fatalf("property %q description is empty", tc.property)
			}
		})
	}

	// Every field is omitempty, so none of them may be required.
	if len(schema.Required) != 0 {
		t.Fatalf("required = %v, want none", schema.Required)
	}
}

func TestBuildAlarmQueryCondition(t *testing.T) {
	timeCtx := TimeContext{
		NowUTC:   time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		Location: time.UTC,
	}

	req := &AlarmQueryRequest{
		Scope:   "Service",
		Keyword: "timeout",
		Tags: []AlarmTag{
			{Key: "level", Value: "critical"},
			{Key: "team", Value: "payments"},
		},
		Start:    "-2h",
		End:      "now",
		PageNum:  2,
		PageSize: 30,
	}

	cond := buildAlarmQueryCondition(req, timeCtx)

	if cond.Scope != api.Scope("Service") {
		t.Fatalf("scope = %q", cond.Scope)
	}
	if cond.Keyword != "timeout" {
		t.Fatalf("keyword = %q", cond.Keyword)
	}
	if cond.Duration == nil {
		t.Fatal("duration is nil")
	}
	if cond.Duration.Start != testTimeStart {
		t.Fatalf("start = %q", cond.Duration.Start)
	}
	if cond.Duration.End != testTimeEnd {
		t.Fatalf("end = %q", cond.Duration.End)
	}
	if cond.Paging == nil || cond.Paging.PageNum == nil || *cond.Paging.PageNum != 2 {
		t.Fatalf("page num = %v", cond.Paging)
	}
	if cond.Paging.PageSize != 30 {
		t.Fatalf("page size = %d", cond.Paging.PageSize)
	}
	if len(cond.Tags) != 2 {
		t.Fatalf("tags len = %d", len(cond.Tags))
	}
	if cond.Tags[0].Key != "level" || cond.Tags[0].Value == nil || *cond.Tags[0].Value != "critical" {
		t.Fatalf("first tag = %+v", cond.Tags[0])
	}
}

func TestBuildAlarmQueryConditionDefaults(t *testing.T) {
	timeCtx := TimeContext{
		NowUTC:   time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		Location: time.UTC,
	}

	cond := buildAlarmQueryCondition(&AlarmQueryRequest{}, timeCtx)

	if cond.Scope != "" {
		t.Fatalf("scope = %q", cond.Scope)
	}
	if cond.Paging == nil || cond.Paging.PageNum == nil || *cond.Paging.PageNum != DefaultPageNum {
		t.Fatalf("default page num = %v", cond.Paging)
	}
	if cond.Paging.PageSize != DefaultPageSize {
		t.Fatalf("default page size = %d", cond.Paging.PageSize)
	}
	if cond.Duration == nil {
		t.Fatal("duration is nil")
	}
	if cond.Duration.End != testTimeEnd {
		t.Fatalf("end = %q", cond.Duration.End)
	}
}
