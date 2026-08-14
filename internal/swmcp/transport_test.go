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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/config"
)

// proxiedHost stands in for the public Host a same-host reverse proxy
// forwards unchanged to a loopback backend.
const proxiedHost = "mcp.example.com"

// transportServer is a transport served over loopback, which is what arms the
// SDK's DNS rebinding protection, paired with the endpoint it answers on.
type transportServer struct {
	*httptest.Server
	endpoint string
}

// url addresses the endpoint the transport serves rather than the listener
// root, so requests take the route production routes them through.
func (s *transportServer) url() string {
	return s.URL + s.endpoint
}

func serveLoopback(t *testing.T, handler http.Handler, endpoint string) *transportServer {
	t.Helper()

	testServer := httptest.NewServer(handler)
	t.Cleanup(testServer.Close)
	return &transportServer{Server: testServer, endpoint: endpoint}
}

// postStatus posts a tools/list call with the given Host and Origin headers,
// omitting Origin when empty, and reports the response status.
func postStatus(t *testing.T, testServer *transportServer, host, origin string) int {
	t.Helper()

	return postStatusTo(t, testServer, testServer.url(), host, origin)
}

func postStatusTo(t *testing.T, testServer *transportServer, url, host, origin string) int {
	t.Helper()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	res, err := testServer.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	return res.StatusCode
}

type transportUnderTest struct {
	cmd  *cobra.Command
	http func() config.HTTPTransportConfig
}

// registeredTransports builds every subcommand the way main.go does, so the
// tests see the state production runs in: the transports share flag names, and
// a shared registry would let the subcommand registered last own them all.
func registeredTransports(t *testing.T) map[string]transportUnderTest {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	sseCmd, sseConfig := newSSECommand()
	streamableCmd, streamableConfig := newStreamableCommand()

	return map[string]transportUnderTest{
		"sse": {
			cmd:  sseCmd,
			http: func() config.HTTPTransportConfig { return sseConfig().HTTPTransportConfig },
		},
		"streamable": {
			cmd:  streamableCmd,
			http: func() config.HTTPTransportConfig { return streamableConfig().HTTPTransportConfig },
		},
	}
}

// The protection must stay on unless an operator opts out, so every transport
// has to register the flag, default it to false, and carry its own value into
// the config its handler is built from.
func TestLocalhostProtectionFlagDefaultsToEnabled(t *testing.T) {
	for name, transport := range registeredTransports(t) {
		t.Run(name, func(t *testing.T) {
			flag := transport.cmd.Flags().Lookup("disable-localhost-protection")
			if flag == nil {
				t.Fatal("disable-localhost-protection flag not registered")
			}
			if flag.DefValue != "false" {
				t.Fatalf("flag default = %q, want %q", flag.DefValue, "false")
			}
			if transport.http().DisableLocalhostProtection {
				t.Fatal("config reports the protection disabled before the flag is set")
			}

			if err := transport.cmd.Flags().Set("disable-localhost-protection", "true"); err != nil {
				t.Fatalf("set flag: %v", err)
			}
			if !transport.http().DisableLocalhostProtection {
				t.Fatal("setting the flag did not reach the transport's config")
			}
		})
	}
}

// Every transport must read its own --allowed-origins; an empty list means the
// CORS allowlist silently degrades to open.
func TestAllowedOriginsReachesEachTransport(t *testing.T) {
	for name, transport := range registeredTransports(t) {
		t.Run(name, func(t *testing.T) {
			if err := transport.cmd.Flags().Set("allowed-origins", allowedOrigin); err != nil {
				t.Fatalf("set flag: %v", err)
			}

			got := transport.http().AllowedOrigins
			if len(got) != 1 || got[0] != allowedOrigin {
				t.Fatalf("allowed origins = %v, want [%s]", got, allowedOrigin)
			}
		})
	}
}

// The shared flags are also configurable through SW_* environment variables,
// which only works while each per-command viper applies the env conventions;
// dropping ConfigureEnv from bindHTTPTransportFlags would silently break it.
func TestEnvVarsReachEachTransport(t *testing.T) {
	t.Setenv("SW_ALLOWED_ORIGINS", allowedOrigin)
	t.Setenv("SW_DISABLE_LOCALHOST_PROTECTION", "true")

	for name, transport := range registeredTransports(t) {
		t.Run(name, func(t *testing.T) {
			got := transport.http()
			if len(got.AllowedOrigins) != 1 || got.AllowedOrigins[0] != allowedOrigin {
				t.Fatalf("allowed origins = %v, want [%s]", got.AllowedOrigins, allowedOrigin)
			}
			if !got.DisableLocalhostProtection {
				t.Fatal("SW_DISABLE_LOCALHOST_PROTECTION did not reach the transport's config")
			}
		})
	}
}

// Setting a flag on one transport must not leak into another.
func TestTransportFlagsAreIndependent(t *testing.T) {
	transports := registeredTransports(t)

	if err := transports["streamable"].cmd.Flags().Set("disable-localhost-protection", "true"); err != nil {
		t.Fatalf("set flag: %v", err)
	}

	if !transports["streamable"].http().DisableLocalhostProtection {
		t.Fatal("streamable did not pick up its own flag")
	}
	if transports["sse"].http().DisableLocalhostProtection {
		t.Fatal("streamable's flag leaked into sse")
	}
}
