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
	"strings"
)

// corsMiddleware adds CORS response headers and enforces origin validation.
// When allowedOrigins is empty, every request with an Origin header is
// reflected back — i.e., CORS is open and all browser origins work.
// When allowedOrigins is non-empty, only listed origins receive CORS headers;
// requests from any other origin receive 403 Forbidden. Use "*" as an entry
// to explicitly allow all origins via the wildcard header.
// defaultAllowedHeaders covers the fixed headers the MCP transports read:
// Mcp-Protocol-Version, Mcp-Method, and Mcp-Name are required by protocol
// 2026-07-28; Mcp-Session-Id and Last-Event-ID are sent by clients on
// earlier protocol revisions.
const defaultAllowedHeaders = "Content-Type, Authorization, Accept, " +
	"Mcp-Protocol-Version, Mcp-Method, Mcp-Name, Mcp-Session-Id, Last-Event-ID"

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if len(allowedOrigins) == 0 || isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				// Protocol 2026-07-28 mirrors tool parameters annotated with
				// x-mcp-header into dynamic Mcp-Param-* request headers, so the
				// allowed set cannot be enumerated statically; reflect whatever
				// the preflight declares. Origin validation above remains the
				// security gate.
				if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
					w.Header().Set("Access-Control-Allow-Headers", requested)
				} else {
					w.Header().Set("Access-Control-Allow-Headers", defaultAllowedHeaders)
				}
				w.Header().Set("Vary", "Origin, Access-Control-Request-Headers")
			} else {
				http.Error(w, "forbidden: origin not allowed", http.StatusForbidden)
				return
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isOriginAllowed reports whether origin is in the allowed list.
// The wildcard "*" matches any origin.
func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

// parseAllowedOrigins splits a comma-separated list of origins.
func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
