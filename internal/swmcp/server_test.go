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

const sessionOAPURL = "http://session-oap:12800/graphql"

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

func TestApplySessionOverridesWithoutSessionLeavesContextUnchanged(t *testing.T) {
	ctx := WithSkyWalkingURLAndInsecure(context.Background(), "http://configured-oap:12800/graphql", false)

	got := applySessionOverrides(ctx)
	if gotURL, _ := got.Value(contextkey.BaseURL{}).(string); gotURL != "http://configured-oap:12800/graphql" {
		t.Fatalf("base URL = %q", gotURL)
	}
}

func TestApplySessionOverridesWithURLOnlyKeepsConfiguredAuth(t *testing.T) {
	ctx := WithSkyWalkingURLAndInsecure(context.Background(), "http://configured-oap:12800/graphql", false)
	ctx = WithSkyWalkingAuth(ctx, "configured-user", "configured-pass")

	session := &Session{}
	session.SetConnection(sessionOAPURL, "", "")
	ctx = WithSession(ctx, session)

	got := applySessionOverrides(ctx)

	gotURL, _ := got.Value(contextkey.BaseURL{}).(string)
	if gotURL != sessionOAPURL {
		t.Fatalf("base URL = %q", gotURL)
	}

	gotUser, _ := got.Value(contextkey.Username{}).(string)
	if gotUser != "configured-user" {
		t.Fatalf("username = %q", gotUser)
	}

	gotPass, _ := got.Value(contextkey.Password{}).(string)
	if gotPass != "configured-pass" {
		t.Fatalf("password = %q", gotPass)
	}
}

func TestEnhanceStdioContextFuncStillAllowsSessionOverride(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "http://configured-oap:12800")

	session := &Session{}
	session.SetConnection(sessionOAPURL, "user", "pass")

	ctx := WithSession(context.Background(), session)
	ctx = applySessionOverrides(WithSkyWalkingURLAndInsecure(ctx, configuredSkyWalkingURL(), false))

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != sessionOAPURL {
		t.Fatalf("base URL = %q", gotURL)
	}

	gotUser, _ := ctx.Value(contextkey.Username{}).(string)
	if gotUser != "user" {
		t.Fatalf("username = %q", gotUser)
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
	if gotURL != "https://configured-oap.example.com/graphql" {
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

	if _, ok := ctx.Value(sessionKey{}).(*Session); !ok {
		t.Fatal("session not attached to stdio context")
	}
}

func TestEnhanceHTTPContextFuncDoesNotUseSWURLHeader(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "http://configured-oap:12800")

	req, err := http.NewRequest(http.MethodPost, "http://client/request", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("SW-URL", "http://attacker.invalid:8080")

	ctx := EnhanceHTTPContextFunc()(context.Background(), req)

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != "http://configured-oap:12800/graphql" {
		t.Fatalf("base URL = %q", gotURL)
	}
}

func TestEnhanceSSEContextFuncDoesNotUseSWURLHeader(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "https://configured-oap.example.com")

	req, err := http.NewRequest(http.MethodGet, "http://client/events", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("SW-URL", "https://attacker.invalid")

	ctx := EnhanceSSEContextFunc()(context.Background(), req)

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != "https://configured-oap.example.com/graphql" {
		t.Fatalf("base URL = %q", gotURL)
	}
}
