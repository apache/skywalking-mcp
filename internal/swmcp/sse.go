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
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/config"
)

func NewSSEServer() *cobra.Command {
	sseCmd := &cobra.Command{
		Use:   "sse",
		Short: "Start SSE server",
		Long:  `Start a server that listens for Server-Sent Events (SSE) on the specified address.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateConfiguredSkyWalkingURL(); err != nil {
				return err
			}

			sseServerConfig := config.SSEServerConfig{
				Address:  viper.GetString("sse-address"),
				BasePath: viper.GetString("base-path"),
			}

			return runSSEServer(context.Background(), &sseServerConfig)
		},
	}

	// Add SSE server specific flags
	sseCmd.Flags().String("sse-address", "localhost:8000",
		"The host and port to start the sse server on")
	sseCmd.Flags().String("base-path", "",
		"Base path for the sse server")
	sseCmd.Flags().String("allowed-origins", "",
		"Comma-separated allowed CORS origins. Empty = open (any origin reflected). Use * for wildcard header.")
	_ = viper.BindPFlag("sse-address", sseCmd.Flags().Lookup("sse-address"))
	_ = viper.BindPFlag("base-path", sseCmd.Flags().Lookup("base-path"))
	_ = viper.BindPFlag("allowed-origins", sseCmd.Flags().Lookup("allowed-origins"))

	return sseCmd
}

// runSSEServer starts a server that listens for Server-Sent Events (SSE) on the specified address.
func runSSEServer(ctx context.Context, cfg *config.SSEServerConfig) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := initLogger(cfg.LogFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	allowedOrigins := parseAllowedOrigins(viper.GetString("allowed-origins"))

	// The HTTP+SSE transport predates streamable HTTP and is kept for clients
	// that still speak it.
	srv := newMCPServer()
	handler := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return srv }, nil)

	basePath := normalizeHTTPPath(cfg.BasePath)
	ssePath := basePath + "/sse"
	mux := http.NewServeMux()
	mux.Handle(basePath+"/", corsMiddleware(allowedOrigins, handler))

	sseServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Starting SkyWalking MCP server using SSE transport listening on http://%s%s\n ", cfg.Address, ssePath)

	errCh := make(chan error, 1)
	go func() {
		if err := sseServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err // bubble up real crashes
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Block until Ctrl-C or an internal error
	select {
	case <-ctx.Done():
		// user hit Ctrl-C
		_, _ = fmt.Fprintln(os.Stderr, "Received shutdown signal, stopping server...")
	case err := <-errCh:
		// HTTP server crashed
		return fmt.Errorf("sse server error: %w", err)
	}

	// Graceful shutdown. SSE streams never become idle, so Shutdown alone
	// would wait out its full deadline whenever a client is still connected;
	// force-close whatever outlives it.
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sseServer.Shutdown(shCtx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Infof("closing remaining SSE connections: %v", err)
			_ = sseServer.Close()
		}
	}

	_, _ = fmt.Fprintln(os.Stderr, "SSE server stopped gracefully")
	return nil
}
