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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	api "skywalking.apache.org/repo/goapi/query"

	"github.com/apache/skywalking-cli/pkg/contextkey"
)

// fakeOAPServer emulates the OAP GraphQL endpoint for trace queries. It routes on
// the query text: the v2-support probe, the v1 basic-traces query, the v1
// single-trace query, and the v2 queryTraces query.
func fakeOAPServer(t *testing.T, v2Supported bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(body.Query, "hasQueryTracesV2Support"):
			fmt.Fprintf(w, `{"data":{"result":%t}}`, v2Supported)
		case strings.Contains(body.Query, "queryBasicTraces"):
			// Three summaries referencing two unique trace IDs (t1 appears twice).
			_, _ = w.Write([]byte(`{"data":{"result":{"traces":[` +
				`{"segmentId":"s1","endpointNames":["/a"],"duration":100,"start":"1000","isError":false,"traceIds":["t1"]},` +
				`{"segmentId":"s2","endpointNames":["/a"],"duration":100,"start":"1000","isError":false,"traceIds":["t1"]},` +
				`{"segmentId":"s3","endpointNames":["/b"],"duration":200,"start":"2000","isError":true,"traceIds":["t2"]}` +
				`]}}}`))
		case strings.Contains(body.Query, "queryTraces("):
			// v2 path.
			_, _ = w.Write([]byte(`{"data":{"result":{"traces":[{"spans":[` +
				`{"traceId":"v2","segmentId":"seg","spanId":0,"parentSpanId":-1,"serviceCode":"svcV2",` +
				`"startTime":10,"endTime":20,"endpointName":"/v2","type":"Entry","isError":false}` +
				`]}],"retrievedTimeRange":{"startTime":0,"endTime":0}}}}`))
		case strings.Contains(body.Query, "queryTrace("):
			traceID, _ := body.Variables["traceId"].(string)
			svc, ep, isErr, st, et := "svcA", "/a", "false", 1000, 1100
			if traceID == "t2" {
				svc, ep, isErr, st, et = "svcB", "/b", "true", 2000, 2200
			}
			fmt.Fprintf(w, `{"data":{"result":{"spans":[`+
				`{"traceId":%q,"segmentId":"seg","spanId":0,"parentSpanId":-1,"serviceCode":%q,`+
				`"serviceInstanceName":"i","startTime":%d,"endTime":%d,"endpointName":%q,"type":"Entry",`+
				`"isError":%s,"refs":[],"tags":[],"logs":[],"attachedEvents":[]}`+
				`]}}}`, traceID, svc, st, et, ep, isErr)
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
		}
	}))
}

// traceTestContext builds a context carrying the base URL and (required) insecure flag
// expected by the skywalking-cli GraphQL client.
func traceTestContext(url string) context.Context {
	ctx := context.WithValue(context.Background(), contextkey.BaseURL{}, url)
	return context.WithValue(ctx, contextkey.Insecure{}, false)
}

func TestSupportsTraceV2(t *testing.T) {
	tsTrue := fakeOAPServer(t, true)
	defer tsTrue.Close()
	if !supportsTraceV2(traceTestContext(tsTrue.URL)) {
		t.Fatal("expected v2 support to be true")
	}

	tsFalse := fakeOAPServer(t, false)
	defer tsFalse.Close()
	if supportsTraceV2(traceTestContext(tsFalse.URL)) {
		t.Fatal("expected v2 support to be false")
	}

	// An unreachable backend must degrade to "not supported" so the v1 path is used.
	if supportsTraceV2(traceTestContext("http://127.0.0.1:1")) {
		t.Fatal("expected v2 support to be false on connection error")
	}
}

func TestQueryTracesAutoUsesV1WhenV2Unsupported(t *testing.T) {
	ts := fakeOAPServer(t, false)
	defer ts.Close()

	list, err := queryTracesAuto(traceTestContext(ts.URL), &api.TraceQueryCondition{})
	if err != nil {
		t.Fatalf("queryTracesAuto returned error: %v", err)
	}

	// Three basic-trace rows collapse to two unique traces (t1 is deduplicated).
	if len(list.Traces) != 2 {
		t.Fatalf("expected 2 deduplicated traces, got %d", len(list.Traces))
	}
	for _, tr := range list.Traces {
		if len(tr.Spans) == 0 {
			t.Fatal("expected each assembled trace to carry its spans from queryTrace")
		}
	}

	// The summary is derived from the fetched spans, including the error state.
	result, err := processTracesResult(&list, ViewSummary, 0)
	if err != nil {
		t.Fatalf("processTracesResult returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	var summary TracesSummary
	if err := json.Unmarshal([]byte(text.Text), &summary); err != nil {
		t.Fatalf("failed to decode summary: %v", err)
	}
	if summary.TotalTraces != 2 {
		t.Fatalf("total_traces = %d, want 2", summary.TotalTraces)
	}
	if summary.ErrorCount != 1 {
		t.Fatalf("error_count = %d, want 1", summary.ErrorCount)
	}
	if summary.SuccessCount != 1 {
		t.Fatalf("success_count = %d, want 1", summary.SuccessCount)
	}
}

// TestQueryTraceV1OmitsDurationArgument guards against re-introducing the queryTrace
// `duration` argument, which does not exist on OAP releases older than 10.3.0 (and is
// BanyanDB-only where it does), and would break the v1 fallback on the exact older,
// non-BanyanDB backends it targets.
func TestQueryTraceV1OmitsDurationArgument(t *testing.T) {
	if strings.Contains(queryTraceV1GQL, "duration") {
		t.Fatal("queryTraceV1GQL must not reference the duration argument (breaks OAP < 10.3.0)")
	}
}

func TestQueryTracesAutoUsesV2WhenSupported(t *testing.T) {
	ts := fakeOAPServer(t, true)
	defer ts.Close()

	list, err := queryTracesAuto(traceTestContext(ts.URL), &api.TraceQueryCondition{})
	if err != nil {
		t.Fatalf("queryTracesAuto returned error: %v", err)
	}
	if len(list.Traces) != 1 {
		t.Fatalf("expected 1 trace from v2 path, got %d", len(list.Traces))
	}
	if len(list.Traces[0].Spans) == 0 || list.Traces[0].Spans[0].ServiceCode != "svcV2" {
		t.Fatalf("expected v2 span with serviceCode svcV2, got %+v", list.Traces[0].Spans)
	}
}
