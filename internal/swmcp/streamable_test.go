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

// serveStreamable serves the mux runStreamableServer builds, over the command's
// own defaults, so the tests reach the endpoint production serves rather than
// the listener root.
func serveStreamable(t *testing.T, httpConfig config.HTTPTransportConfig) *transportServer {
	t.Helper()

	_, defaults := newStreamableCommand()
	cfg := defaults()
	cfg.HTTPTransportConfig = httpConfig

	mux, endpoint := newStreamableMux(&cfg)
	return serveLoopback(t, mux, endpoint)
}

func TestStreamableRejectsProxiedHostByDefault(t *testing.T) {
	testServer := serveStreamable(t, config.HTTPTransportConfig{})

	if got := postStatus(t, testServer, proxiedHost, ""); got != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for a non-localhost Host on a loopback listener", got)
	}
}

func TestStreamableAcceptsProxiedHostWhenProtectionDisabled(t *testing.T) {
	testServer := serveStreamable(t, config.HTTPTransportConfig{DisableLocalhostProtection: true})

	if got := postStatus(t, testServer, proxiedHost, ""); got != http.StatusOK {
		t.Fatalf("status = %d, want 200 once localhost protection is disabled", got)
	}
}

// A direct local client is unaffected either way, so the protection must not
// be the reason a plain loopback request fails.
func TestStreamableAcceptsLoopbackHost(t *testing.T) {
	testServer := serveStreamable(t, config.HTTPTransportConfig{})

	if got := postStatus(t, testServer, testServer.Listener.Addr().String(), ""); got != http.StatusOK {
		t.Fatalf("status = %d, want 200 for a loopback Host", got)
	}
}

// The mux has to serve the configured endpoint and only that, so a request to
// another path is a 404 rather than a catch-all hit.
func TestStreamableServesConfiguredEndpoint(t *testing.T) {
	cfg := config.StreamableServerConfig{EndpointPath: "/custom-mcp"}
	mux, endpoint := newStreamableMux(&cfg)
	if endpoint != "/custom-mcp" {
		t.Fatalf("endpoint = %q, want /custom-mcp", endpoint)
	}

	testServer := serveLoopback(t, mux, endpoint)
	host := testServer.Listener.Addr().String()

	if got := postStatus(t, testServer, host, ""); got != http.StatusOK {
		t.Fatalf("status = %d on the configured endpoint, want 200", got)
	}
	if got := postStatusTo(t, testServer, testServer.URL+"/", host, ""); got != http.StatusNotFound {
		t.Fatalf("status = %d on the listener root, want 404", got)
	}
}

// The endpoint is mounted in both forms, so a client that appends a slash to
// the configured path still reaches the server.
func TestStreamableServesTrailingSlashEndpoint(t *testing.T) {
	testServer := serveStreamable(t, config.HTTPTransportConfig{})

	url := testServer.url() + "/"
	if got := postStatusTo(t, testServer, url, testServer.Listener.Addr().String(), ""); got != http.StatusOK {
		t.Fatalf("status = %d for %s, want 200", got, url)
	}
}

// The README tells operators to pair the two flags, because disabling the
// protection removes the Host check and leaves the origin allowlist as the only
// barrier for a browser on the proxy host. Both halves have to carry weight:
// the allowed origin only gets through with the protection off, and the
// disallowed one only gets rejected while the allowlist is enforced.
func TestProxiedHostStillHonorsOriginAllowlist(t *testing.T) {
	testServer := serveStreamable(t, config.HTTPTransportConfig{
		AllowedOrigins:             []string{allowedOrigin},
		DisableLocalhostProtection: true,
	})

	if got := postStatus(t, testServer, proxiedHost, allowedOrigin); got != http.StatusOK {
		t.Fatalf("allowed origin status = %d, want 200", got)
	}
	if got := postStatus(t, testServer, proxiedHost, "https://evil.example"); got != http.StatusForbidden {
		t.Fatalf("disallowed origin status = %d, want 403", got)
	}
}
