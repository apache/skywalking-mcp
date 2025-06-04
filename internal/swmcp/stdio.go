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
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/apache/skywalking-cli/pkg/contextkey"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/config"
	"github.com/apache/skywalking-mcp/internal/tools"
)

func NewStdioServer() *cobra.Command {
	return &cobra.Command{
		Use:   "stdio",
		Short: "Start stdio server",
		Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			url := viper.GetString("url")
			if url == "" {
				return errors.New("SW_URL must be specified")
			}

			stdioServerConfig := config.StdioServerConfig{
				URL:         url,
				ReadOnly:    viper.GetBool("read-only"),
				LogFilePath: viper.GetString("log-file"),
				LogCommands: viper.GetBool("log-command"),
			}

			return runStdioServer(context.Background(), &stdioServerConfig)
		},
	}
}

// runStdioServer starts a standard input/output server for the MCP protocol.
func runStdioServer(ctx context.Context, cfg *config.StdioServerConfig) error {
	slog.Info("Start a server that communicates via standard input/output streams using JSON-RPC messages.")
	// Handle SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	stdioServer := server.NewStdioServer(newMcpServer())

	logger, err := initLogger(cfg.LogFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	stdLogger := log.New(logger.Writer(), "swmcp-stdio-server", 0)
	stdioServer.SetErrorLogger(stdLogger)
	stdioServer.SetContextFunc(EnhanceStdioContextFunc())

	// Start listening for messages
	errC := make(chan error, 1)
	go func() {
		in, out := io.Reader(os.Stdin), io.Writer(os.Stdout)

		if cfg.LogCommands {
			loggedIO := tools.NewIOLogger(in, out, logger)
			in, out = loggedIO, loggedIO
		}

		errC <- stdioServer.Listen(ctx, in, out)
	}()

	// Output github-mcp-server string
	_, _ = fmt.Fprintf(os.Stderr, "SkyWalking MCP Server running on stdio\n")

	// Wait for shutdown signal
	select {
	case <-ctx.Done():
		logger.Infof("shutting down server...")
	case err := <-errC:
		if err != nil {
			return fmt.Errorf("error running server: %w", err)
		}
	}

	return nil
}

var ExtractSWURLFromCfg server.StdioContextFunc = func(ctx context.Context) context.Context {
	urlStr := viper.GetString("url")
	if urlStr == "" {
		urlStr = config.DefaultSWURL
	}

	// we need to ensure the URL ends with "/graphql"
	if !strings.HasSuffix(urlStr, "/graphql") {
		urlStr = strings.TrimRight(urlStr, "/") + "/graphql"
	}
	return WithSkyWalkingURLAndInsecure(ctx, urlStr)
}

var ExtractSWURLFromHeaders server.SSEContextFunc = func(ctx context.Context, req *http.Request) context.Context {
	urlStr := req.Header.Get("SW-URL")
	if urlStr == "" {
		urlStr = viper.GetString("url")
		if urlStr == "" {
			urlStr = config.DefaultSWURL
		}
	}

	// we need to ensure the URL ends with "/graphql"
	if !strings.HasSuffix(urlStr, "/graphql") {
		urlStr = strings.TrimRight(urlStr, "/") + "/graphql"
	}
	return WithSkyWalkingURLAndInsecure(ctx, urlStr)
}

func EnhanceStdioContextFuncs(funcs ...server.StdioContextFunc) server.StdioContextFunc {
	return func(ctx context.Context) context.Context {
		for _, f := range funcs {
			ctx = f(ctx)
		}
		return ctx
	}
}

func EnhanceSSEContextFuncs(funcs ...server.SSEContextFunc) server.SSEContextFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		for _, f := range funcs {
			ctx = f(ctx, r)
		}
		return ctx
	}
}

// WithSkyWalkingURLAndInsecure adds the SkyWalking URL and Insecure to the context.
func WithSkyWalkingURLAndInsecure(ctx context.Context, url string) context.Context {
	ctx = context.WithValue(ctx, contextkey.BaseURL{}, url)
	ctx = context.WithValue(ctx, contextkey.Insecure{}, false)
	return ctx
}

// EnhanceStdioContextFunc returns a StdioContextFunc that composes all the provided StdioContextFuncs.
func EnhanceStdioContextFunc() server.StdioContextFunc {
	return EnhanceStdioContextFuncs(ExtractSWURLFromCfg)
}

// EnhanceHTTPContextFunc returns a SSEContextFunc that composes all the provided HTTPContextFuncs.
func EnhanceHTTPContextFunc() server.SSEContextFunc {
	return EnhanceSSEContextFuncs(ExtractSWURLFromHeaders)
}
