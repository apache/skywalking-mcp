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

package swmcp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	allowedOrigin = "http://allowed.example.com"
	openOrigin    = "http://any.example.com"
)

// sentinelHandler is a simple handler that records whether it was called.
type sentinelHandler struct{ called bool }

func (s *sentinelHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	s.called = true
	w.WriteHeader(http.StatusOK)
}

func corsRequest(method, origin string, allowedOrigins []string) (*httptest.ResponseRecorder, *sentinelHandler) {
	sentinel := &sentinelHandler{}
	handler := corsMiddleware(allowedOrigins, sentinel)

	req := httptest.NewRequest(method, "/mcp", http.NoBody)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr, sentinel
}

// --- empty allowlist (open CORS) ---

func TestCORSEmptyAllowlistNoOriginHeader(t *testing.T) {
	rr, sentinel := corsRequest(http.MethodPost, "", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !sentinel.called {
		t.Fatal("next handler not called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACAO header should be absent without Origin, got %q", got)
	}
}

func TestCORSEmptyAllowlistReflectsAnyOrigin(t *testing.T) {
	rr, sentinel := corsRequest(http.MethodPost, openOrigin, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !sentinel.called {
		t.Fatal("next handler not called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != openOrigin {
		t.Fatalf("ACAO = %q, want reflected origin", got)
	}
	if got := rr.Header().Get("Vary"); got != "Origin, Access-Control-Request-Headers" {
		t.Fatalf("Vary = %q, want Origin and Access-Control-Request-Headers", got)
	}
}

// --- non-empty allowlist ---

func TestCORSAllowedOriginReceivesHeaders(t *testing.T) {
	allowed := []string{allowedOrigin}
	rr, sentinel := corsRequest(http.MethodPost, allowedOrigin, allowed)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !sentinel.called {
		t.Fatal("next handler not called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("ACAO = %q, want reflected origin", got)
	}
}

func TestCORSDisallowedOriginRejects(t *testing.T) {
	allowed := []string{allowedOrigin}
	rr, sentinel := corsRequest(http.MethodPost, "http://attacker.invalid", allowed)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if sentinel.called {
		t.Fatal("next handler must not be called for disallowed origin")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACAO header should be absent for rejected origin, got %q", got)
	}
}

func TestCORSNoOriginHeaderWithAllowlist(t *testing.T) {
	allowed := []string{allowedOrigin}
	rr, sentinel := corsRequest(http.MethodPost, "", allowed)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (non-browser requests without Origin pass through)", rr.Code)
	}
	if !sentinel.called {
		t.Fatal("next handler not called")
	}
}

// --- wildcard entry ---

func TestCORSWildcardEntryReflectsOrigin(t *testing.T) {
	allowed := []string{"*"}
	rr, sentinel := corsRequest(http.MethodPost, "http://anything.example.com", allowed)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if !sentinel.called {
		t.Fatal("next handler not called")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://anything.example.com" {
		t.Fatalf("ACAO = %q, want reflected origin for wildcard entry", got)
	}
}

// --- preflight (OPTIONS) ---

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	allowed := []string{allowedOrigin}
	rr, sentinel := corsRequest(http.MethodOptions, allowedOrigin, allowed)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if sentinel.called {
		t.Fatal("next handler must not be called for preflight")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Fatalf("ACAO = %q, want reflected origin on preflight", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Access-Control-Allow-Methods should be set on preflight")
	}
}

func TestCORSPreflightDisallowedOriginRejects(t *testing.T) {
	allowed := []string{allowedOrigin}
	rr, sentinel := corsRequest(http.MethodOptions, "http://attacker.invalid", allowed)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	if sentinel.called {
		t.Fatal("next handler must not be called")
	}
}

func TestCORSPreflightEmptyAllowlist(t *testing.T) {
	rr, sentinel := corsRequest(http.MethodOptions, openOrigin, nil)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if sentinel.called {
		t.Fatal("next handler must not be called for preflight")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != openOrigin {
		t.Fatalf("ACAO = %q, want reflected origin on open-CORS preflight", got)
	}
}

// Protocol 2026-07-28 clients send Mcp-Param-* headers derived from tool
// parameters, so a preflight declaring them must be allowed as-is.
func TestCORSPreflightReflectsRequestedHeaders(t *testing.T) {
	sentinel := &sentinelHandler{}
	handler := corsMiddleware(nil, sentinel)

	req := httptest.NewRequest(http.MethodOptions, "/mcp", http.NoBody)
	req.Header.Set("Origin", openOrigin)
	req.Header.Set("Access-Control-Request-Headers", "mcp-method, mcp-name, mcp-param-region")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "mcp-method, mcp-name, mcp-param-region" {
		t.Fatalf("Allow-Headers = %q, want the requested headers reflected", got)
	}
}

func TestCORSPreflightDefaultAllowsMCPHeaders(t *testing.T) {
	rr, _ := corsRequest(http.MethodOptions, openOrigin, nil)

	got := rr.Header().Get("Access-Control-Allow-Headers")
	for _, header := range []string{"Mcp-Protocol-Version", "Mcp-Method", "Mcp-Name", "Mcp-Session-Id"} {
		if !strings.Contains(got, header) {
			t.Fatalf("Allow-Headers = %q, missing %q", got, header)
		}
	}
}

// --- parseAllowedOrigins ---

func TestParseAllowedOrigins(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "single", in: "http://a.example.com", want: []string{"http://a.example.com"}},
		{name: "multiple", in: "http://a.example.com,https://b.example.com", want: []string{"http://a.example.com", "https://b.example.com"}},
		{name: "trims spaces", in: " http://a.example.com , https://b.example.com ", want: []string{"http://a.example.com", "https://b.example.com"}},
		{name: "wildcard", in: "*", want: []string{"*"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAllowedOrigins(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d: %v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
