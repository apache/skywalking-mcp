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

	api "skywalking.apache.org/repo/goapi/query"
)

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
