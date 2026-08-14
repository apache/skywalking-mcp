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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-cli/pkg/contextkey"

	"github.com/apache/skywalking-mcp/internal/config"
	"github.com/apache/skywalking-mcp/internal/prompts"
	"github.com/apache/skywalking-mcp/internal/tools"
)

// newMCPServer creates a new MCP server with all tools and prompts registered.
func newMCPServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "skywalking-mcp", Version: "0.1.0"}, nil)
	s.AddReceivingMiddleware(skyWalkingContextMiddleware())

	tools.AddTraceTools(s)
	tools.AddLogTools(s)
	tools.AddMQETools(s)
	tools.AddMetadataTools(s)
	tools.AddEventTools(s)
	tools.AddAlarmTools(s)
	tools.AddTopologyTools(s)
	prompts.AddSkyWalkingPrompts(s)
	return s
}

// skyWalkingContextMiddleware injects the configured SkyWalking endpoint and
// credentials into every incoming request. It replaces the per-transport
// context funcs, which all resolved the same global configuration and ignored
// their *http.Request argument.
func skyWalkingContextMiddleware() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ctx = WithSkyWalkingURLAndInsecure(ctx, configuredSkyWalkingURL(), viper.GetBool("insecure"))
			ctx = withConfiguredAuth(ctx)
			return next(ctx, method, req)
		}
	}
}

func initLogger(logFilePath string) (*logrus.Logger, error) {
	if logFilePath == "" {
		return logrus.New(), nil
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logrusLogger := logrus.New()
	logrusLogger.SetFormatter(&logrus.TextFormatter{})
	logrusLogger.SetLevel(logrus.DebugLevel)
	logrusLogger.SetOutput(file)

	return logrusLogger, nil
}

// WithSkyWalkingURLAndInsecure adds SkyWalking URL and insecure flag to the context.
// This ensures all downstream requests will have contextkey.BaseURL{} and contextkey.Insecure{}.
func WithSkyWalkingURLAndInsecure(ctx context.Context, url string, insecure bool) context.Context {
	ctx = context.WithValue(ctx, contextkey.BaseURL{}, url)
	ctx = context.WithValue(ctx, contextkey.Insecure{}, insecure)
	return ctx
}

// WithSkyWalkingAuth adds username and password to the context for basic auth.
func WithSkyWalkingAuth(ctx context.Context, username, password string) context.Context {
	ctx = context.WithValue(ctx, contextkey.Username{}, username)
	ctx = context.WithValue(ctx, contextkey.Password{}, password)
	return ctx
}

// configuredSkyWalkingURL returns the configured SkyWalking OAP URL.
// The value is sourced from the CLI/config binding for `--sw-url`,
// falling back to the built-in default when unset.
func configuredSkyWalkingURL() string {
	resolvedURL, err := resolvedConfiguredSkyWalkingURL()
	if err != nil {
		logrus.WithError(err).Warn("invalid SkyWalking OAP URL configuration; falling back to default URL")
		return config.DefaultSWURL
	}
	return resolvedURL
}

func resolvedConfiguredSkyWalkingURL() (string, error) {
	urlStr := viper.GetString("url")
	if urlStr == "" {
		urlStr = config.DefaultSWURL
	}
	return tools.NormalizeOAPURL(urlStr)
}

func validateConfiguredSkyWalkingURL() error {
	_, err := resolvedConfiguredSkyWalkingURL()
	return err
}

// disableLocalhostProtectionUsage documents the flag shared by the HTTP
// transports. The SDK rejects a loopback request whose Host header is not a
// loopback name, which a same-host reverse proxy preserving the public Host
// looks exactly like.
const disableLocalhostProtectionUsage = "Disable DNS rebinding protection, which rejects loopback requests " +
	"carrying a non-localhost Host header. Enable only behind a trusted reverse proxy that preserves the public Host."

const allowedOriginsUsage = "Comma-separated allowed CORS origins. " +
	"Empty = open (any origin reflected). Use * for wildcard header."

// ConfigureEnv applies the environment variable conventions shared by the root
// command's viper and the per-transport ones.
func ConfigureEnv(v *viper.Viper) {
	v.SetEnvPrefix("SW")
	v.AutomaticEnv()
	// All fields with . or - will be replaced with _ for ENV vars
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
}

// bindHTTPTransportFlags registers the flags every HTTP transport accepts and
// binds them to a viper private to that subcommand. The transports share flag
// names, and viper keeps a single pflag binding per key, so binding them on the
// global viper lets the subcommand registered last own every transport's
// values.
func bindHTTPTransportFlags(cmd *cobra.Command) *viper.Viper {
	v := viper.New()
	ConfigureEnv(v)

	cmd.Flags().String("allowed-origins", "", allowedOriginsUsage)
	cmd.Flags().Bool("disable-localhost-protection", false, disableLocalhostProtectionUsage)
	_ = v.BindPFlag("allowed-origins", cmd.Flags().Lookup("allowed-origins"))
	_ = v.BindPFlag("disable-localhost-protection", cmd.Flags().Lookup("disable-localhost-protection"))

	return v
}

// httpTransportConfigFrom reads the shared HTTP transport settings.
func httpTransportConfigFrom(v *viper.Viper) config.HTTPTransportConfig {
	return config.HTTPTransportConfig{
		AllowedOrigins:             parseAllowedOrigins(v.GetString("allowed-origins")),
		DisableLocalhostProtection: v.GetBool("disable-localhost-protection"),
	}
}

// normalizeHTTPPath applies the same normalization the previous SDK applied to
// transport path flags: "mcp" and "/mcp/" both become "/mcp"; "" and "/"
// become "". Without it, a value like "mcp" is a host pattern to ServeMux and
// "" makes ServeMux panic.
func normalizeHTTPPath(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed
}

// resolveEnvVar resolves a value that may contain an environment variable reference
// in the form ${VAR_NAME}. If the value matches this pattern, it returns the
// environment variable's value. Otherwise, it returns the raw value as-is.
func resolveEnvVar(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "${") && strings.HasSuffix(trimmed, "}") {
		envName := trimmed[2 : len(trimmed)-1]
		resolved, ok := os.LookupEnv(envName)
		if !ok {
			logrus.Warnf("environment variable %q is referenced but not set", envName)
			return ""
		}
		return resolved
	}
	return value
}

// configuredAuth returns the configured username and password from CLI flags or env vars.
func configuredAuth() (username, password string) {
	return resolveEnvVar(viper.GetString("username")), resolveEnvVar(viper.GetString("password"))
}

// withConfiguredAuth injects the configured auth credentials into the context if present.
func withConfiguredAuth(ctx context.Context) context.Context {
	username, password := configuredAuth()
	if username != "" {
		ctx = WithSkyWalkingAuth(ctx, username, password)
	}
	return ctx
}
