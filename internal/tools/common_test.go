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

func TestFinalizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "adds graphql suffix", in: "http://localhost:12800", want: "http://localhost:12800/graphql"},
		{name: "trims trailing slash", in: "http://localhost:12800/", want: "http://localhost:12800/graphql"},
		{name: "keeps existing graphql", in: "http://localhost:12800/graphql", want: "http://localhost:12800/graphql"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := FinalizeURL(tc.in); got != tc.want {
				t.Fatalf("FinalizeURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseTimezoneOffset(t *testing.T) {
	loc, ok := parseTimezoneOffset("+0830")
	if !ok {
		t.Fatal("expected timezone to parse")
	}
	if got := loc.String(); got != "+0830" {
		t.Fatalf("location name = %q", got)
	}

	_, ok = parseTimezoneOffset("UTC")
	if ok {
		t.Fatal("expected invalid timezone offset to fail")
	}
}

func TestParseDurationWithContextRelativeDuration(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	timeCtx := TimeContext{NowUTC: now, Location: time.UTC}

	got := ParseDurationWithContext("-2h", false, timeCtx)

	if got.Start != "2026-03-31 1000" {
		t.Fatalf("start = %q", got.Start)
	}
	if got.End != "2026-03-31 1200" {
		t.Fatalf("end = %q", got.End)
	}
	if got.Step != api.StepMinute {
		t.Fatalf("step = %q", got.Step)
	}
}

func TestParseDurationWithContextLegacyDays(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	timeCtx := TimeContext{NowUTC: now, Location: time.UTC}

	got := ParseDurationWithContext("7d", true, timeCtx)

	if got.Start != "2026-03-24" {
		t.Fatalf("start = %q", got.Start)
	}
	if got.End != "2026-03-31" {
		t.Fatalf("end = %q", got.End)
	}
	if got.Step != api.StepDay {
		t.Fatalf("step = %q", got.Step)
	}
	if got.ColdStage == nil || !*got.ColdStage {
		t.Fatal("expected cold stage to be true")
	}
}

func TestBuildPaginationDefaultsAndCustomValues(t *testing.T) {
	gotDefault := BuildPagination(0, 0)
	if gotDefault.PageNum == nil || *gotDefault.PageNum != DefaultPageNum {
		t.Fatalf("default page num = %v", gotDefault.PageNum)
	}
	if gotDefault.PageSize != DefaultPageSize {
		t.Fatalf("default page size = %d", gotDefault.PageSize)
	}

	gotCustom := BuildPagination(3, 50)
	if gotCustom.PageNum == nil || *gotCustom.PageNum != 3 {
		t.Fatalf("custom page num = %v", gotCustom.PageNum)
	}
	if gotCustom.PageSize != 50 {
		t.Fatalf("custom page size = %d", gotCustom.PageSize)
	}
}

func TestBuildDurationWithContextParsesAbsoluteTimes(t *testing.T) {
	timeCtx := TimeContext{
		NowUTC:   time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC),
		Location: time.FixedZone("+0800", 8*3600),
	}

	got := BuildDurationWithContext("2026-03-31 18:00:00", "2026-03-31 20:00:00", "", false, 30, timeCtx)

	if got.Start != "2026-03-31 1000" {
		t.Fatalf("start = %q", got.Start)
	}
	if got.End != "2026-03-31 1200" {
		t.Fatalf("end = %q", got.End)
	}
	if got.Step != api.StepMinute {
		t.Fatalf("step = %q", got.Step)
	}
}
