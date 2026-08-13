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
	"testing"

	"github.com/apache/skywalking-mcp/internal/config"
)

// serveSSE serves the mux runSSEServer builds, over the command's own defaults,
// for the same reason serveStreamable does.
func serveSSE(t *testing.T, httpConfig config.HTTPTransportConfig) *transportServer {
	t.Helper()

	_, defaults := newSSECommand()
	cfg := defaults()
	cfg.HTTPTransportConfig = httpConfig

	mux, ssePath := newSSEMux(&cfg)
	return serveLoopback(t, mux, ssePath)
}

// The base path has to reach the routes, so the SSE endpoint sits under it and
// nothing answers outside it.
func TestSSEServesConfiguredBasePath(t *testing.T) {
	cfg := config.SSEServerConfig{BasePath: "/custom"}
	mux, ssePath := newSSEMux(&cfg)
	if ssePath != "/custom/sse" {
		t.Fatalf("sse path = %q, want /custom/sse", ssePath)
	}

	testServer := serveLoopback(t, mux, ssePath)
	host := testServer.Listener.Addr().String()

	if got := postStatus(t, testServer, host, ""); got == http.StatusNotFound {
		t.Fatal("status = 404 on the configured base path, want the handler to answer")
	}
	if got := postStatusTo(t, testServer, testServer.URL+"/", host, ""); got != http.StatusNotFound {
		t.Fatalf("status = %d on the listener root, want 404", got)
	}
}

func TestSSERejectsProxiedHostByDefault(t *testing.T) {
	testServer := serveSSE(t, config.HTTPTransportConfig{})

	if got := postStatus(t, testServer, proxiedHost, ""); got != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for a non-localhost Host on a loopback listener", got)
	}
}

// Without a session the SSE handler rejects the POST, but a 403 would mean the
// rebinding check fired instead — that is the property under test.
func TestSSEAcceptsProxiedHostWhenProtectionDisabled(t *testing.T) {
	testServer := serveSSE(t, config.HTTPTransportConfig{DisableLocalhostProtection: true})

	if got := postStatus(t, testServer, proxiedHost, ""); got == http.StatusForbidden {
		t.Fatal("status = 403, want the request to get past localhost protection")
	}
}

// The sse transport must apply its origin allowlist too; it silently did not
// before the per-transport config landed. The allowed origin must also get
// through: without a session the SSE handler still rejects the POST, but a 403
// would mean the allowlist over-blocked it.
func TestSSEHonorsOriginAllowlist(t *testing.T) {
	testServer := serveSSE(t, config.HTTPTransportConfig{AllowedOrigins: []string{allowedOrigin}})
	host := testServer.Listener.Addr().String()

	if got := postStatus(t, testServer, host, allowedOrigin); got == http.StatusForbidden {
		t.Fatal("allowed origin status = 403, want the request past the origin allowlist")
	}
	if got := postStatus(t, testServer, host, "https://evil.example"); got != http.StatusForbidden {
		t.Fatalf("disallowed origin status = %d, want 403", got)
	}
}
