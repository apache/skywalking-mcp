package swmcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/apache/skywalking-cli/pkg/contextkey"
	"github.com/spf13/viper"
)

func TestEnhanceHTTPContextFuncIgnoresSWURLHeader(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "http://configured-oap:12800")

	req, err := http.NewRequest(http.MethodGet, "http://client/request", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("SW-URL", "http://attacker.invalid:8080")

	ctx := EnhanceHTTPContextFunc()(context.Background(), req)
	got, _ := ctx.Value(contextkey.BaseURL{}).(string)
	want := "http://configured-oap:12800/graphql"
	if got != want {
		t.Fatalf("base URL = %q, want %q", got, want)
	}
}

func TestEnhanceSSEContextFuncIgnoresSWURLHeaderAndKeepsConfiguredAuth(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "https://configured-oap.example.com")
	viper.Set("username", "skywalking-user")
	viper.Set("password", "skywalking-pass")

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

	gotUser, _ := ctx.Value(contextkey.Username{}).(string)
	if gotUser != "skywalking-user" {
		t.Fatalf("username = %q", gotUser)
	}

	gotPass, _ := ctx.Value(contextkey.Password{}).(string)
	if gotPass != "skywalking-pass" {
		t.Fatalf("password = %q", gotPass)
	}
}

func TestEnhanceStdioContextFuncStillAllowsSessionOverride(t *testing.T) {
	t.Cleanup(viper.Reset)
	viper.Set("url", "http://configured-oap:12800")

	session := &Session{}
	session.SetConnection("http://session-oap:12800/graphql", "user", "pass")

	ctx := WithSession(context.Background(), session)
	ctx = applySessionOverrides(WithSkyWalkingURLAndInsecure(ctx, configuredSkyWalkingURL(), false))

	gotURL, _ := ctx.Value(contextkey.BaseURL{}).(string)
	if gotURL != "http://session-oap:12800/graphql" {
		t.Fatalf("base URL = %q", gotURL)
	}

	gotUser, _ := ctx.Value(contextkey.Username{}).(string)
	if gotUser != "user" {
		t.Fatalf("username = %q", gotUser)
	}
}
