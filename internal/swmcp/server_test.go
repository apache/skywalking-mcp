// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package swmcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/apache/skywalking-cli/pkg/contextkey"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/config"
)

const (
	configuredHTTPOAPURL  = "http://configured-oap:12800/graphql"
	configuredHTTPSOAPURL = "https://configured-oap.example.com/graphql"
)

func TestConfiguredSkyWalkingURLUsesDefaultWhenUnset(t *testing.T) {
	t.Cleanup(viper.Reset)

	got := configuredSkyWalkingURL()
	if got != config.DefaultSWURL {
		t.Fatalf("configuredSkyWalkingURL() = %q, want %q", got, config.DefaultSWURL)
	}
}

func TestConfiguredSkyWalkingURLFinalizesConfiguredValue(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "https://configured-oap.example.com:12800/")

	got := configuredSkyWalkingURL()
	want := "https://configured-oap.example.com:12800/graphql"
	if got != want {
		t.Fatalf("configuredSkyWalkingURL() = %q, want %q", got, want)
	}
}

func TestResolveEnvVar(t *testing.T) {
	t.Setenv("SW_TEST_SECRET", "resolved-secret")

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "raw", value: "raw-value", want: "raw-value"},
		{name: "env", value: "${SW_TEST_SECRET}", want: "resolved-secret"},
		{name: "trimmed env", value: " ${SW_TEST_SECRET} ", want: "resolved-secret"},
		{name: "missing env", value: "${SW_TEST_MISSING}", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveEnvVar(tc.value); got != tc.want {
				t.Fatalf("resolveEnvVar(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestWithConfiguredAuth(t *testing.T) {
	t.Cleanup(viper.Reset)
	t.Setenv("SW_TEST_USER", "env-user")
	t.Setenv("SW_TEST_PASS", "env-pass")
	viper.Set("username", "${SW_TEST_USER}")
	viper.Set("password", "${SW_TEST_PASS}")

	ctx := withConfiguredAuth(context.Background())

	gotUser, _ := ctx.Value(contextkey.Username{}).(string)
	if gotUser != "env-user" {
		t.Fatalf("username = %q, want %q", gotUser, "env-user")
	}

	gotPass, _ := ctx.Value(contextkey.Password{}).(string)
	if gotPass != "env-pass" {
		t.Fatalf("password = %q, want %q", gotPass, "env-pass")
	}
}

func TestWithConfiguredAuthSkipsEmptyUsername(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("password", "password-only")

	ctx := withConfiguredAuth(context.Background())

	if got, ok := ctx.Value(contextkey.Username{}).(string); ok || got != "" {
		t.Fatalf("username unexpectedly set to %q", got)
	}
	if got, ok := ctx.Value(contextkey.Password{}).(string); ok || got != "" {
		t.Fatalf("password unexpectedly set to %q", got)
	}
}

func TestEnhanceStdioContextFuncUsesConfiguredURLAndAuth(t *testing.T) {
	t.Cleanup(viper.Reset)
	t.Setenv("SW_STDIO_PASS", "stdio-pass")
	viper.Set("url", "https://configured-oap.example.com")
	viper.Set("username", "stdio-user")
	viper.Set("password", "${SW_STDIO_PASS}")

	ctx := EnhanceStdioContextFunc()(context.Background())

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != configuredHTTPSOAPURL {
		t.Fatalf("base URL = %q", gotURL)
	}

	gotUser, _ := ctx.Value(contextkey.Username{}).(string)
	if gotUser != "stdio-user" {
		t.Fatalf("username = %q", gotUser)
	}

	gotPass, _ := ctx.Value(contextkey.Password{}).(string)
	if gotPass != "stdio-pass" {
		t.Fatalf("password = %q", gotPass)
	}
}

func TestEnhanceHTTPContextFuncDoesNotUseSWURLHeader(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "http://configured-oap:12800")

	req, err := http.NewRequest(http.MethodPost, "http://client/request", http.NoBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("SW-URL", "http://attacker.invalid:8080")

	ctx := EnhanceHTTPContextFunc()(context.Background(), req)

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != configuredHTTPOAPURL {
		t.Fatalf("base URL = %q", gotURL)
	}
}

func TestEnhanceSSEContextFuncDoesNotUseSWURLHeader(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "https://configured-oap.example.com")

	req, err := http.NewRequest(http.MethodGet, "http://client/events", http.NoBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("SW-URL", "https://attacker.invalid")

	ctx := EnhanceSSEContextFunc()(context.Background(), req)

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != configuredHTTPSOAPURL {
		t.Fatalf("base URL = %q", gotURL)
	}
}

func TestInsecureFlagDefaultsToFalse(t *testing.T) {
	t.Cleanup(viper.Reset)

	req, _ := http.NewRequest(http.MethodGet, "http://client/events", http.NoBody)

	for name, ctx := range map[string]context.Context{
		"stdio":      EnhanceStdioContextFunc()(context.Background()),
		"sse":        EnhanceSSEContextFunc()(context.Background(), req),
		"streamable": EnhanceHTTPContextFunc()(context.Background(), req),
	} {
		insecure, _ := ctx.Value(contextkey.Insecure{}).(bool)
		if insecure {
			t.Errorf("%s: contextkey.Insecure{} should default to false", name)
		}
	}
}

func TestInsecureFlagPropagatedToContext(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("insecure", true)

	req, _ := http.NewRequest(http.MethodGet, "http://client/events", http.NoBody)

	for name, ctx := range map[string]context.Context{
		"stdio":      EnhanceStdioContextFunc()(context.Background()),
		"sse":        EnhanceSSEContextFunc()(context.Background(), req),
		"streamable": EnhanceHTTPContextFunc()(context.Background(), req),
	} {
		insecure, ok := ctx.Value(contextkey.Insecure{}).(bool)
		if !ok || !insecure {
			t.Errorf("%s: contextkey.Insecure{} should be true when viper insecure=true", name)
		}
	}
}
