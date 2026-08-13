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

	"github.com/apache/skywalking-mcp/internal/config"
)

func NewSSEServer() *cobra.Command {
	cmd, _ := newSSECommand()
	return cmd
}

// newSSECommand also returns the accessor for the configuration the command
// runs with, so callers observe exactly what RunE consumes.
func newSSECommand() (cmd *cobra.Command, cfg func() config.SSEServerConfig) {
	sseCmd := &cobra.Command{
		Use:   "sse",
		Short: "Start SSE server",
		Long:  `Start a server that listens for Server-Sent Events (SSE) on the specified address.`,
	}

	// Add SSE server specific flags
	v := bindHTTPTransportFlags(sseCmd)
	sseCmd.Flags().String("sse-address", "localhost:8000",
		"The host and port to start the sse server on")
	sseCmd.Flags().String("base-path", "",
		"Base path for the sse server")
	_ = v.BindPFlag("sse-address", sseCmd.Flags().Lookup("sse-address"))
	_ = v.BindPFlag("base-path", sseCmd.Flags().Lookup("base-path"))

	sseConfig := func() config.SSEServerConfig {
		return config.SSEServerConfig{
			Address:             v.GetString("sse-address"),
			BasePath:            v.GetString("base-path"),
			HTTPTransportConfig: httpTransportConfigFrom(v),
		}
	}

	sseCmd.RunE = func(_ *cobra.Command, _ []string) error {
		if err := validateConfiguredSkyWalkingURL(); err != nil {
			return err
		}

		sseServerConfig := sseConfig()

		return runSSEServer(context.Background(), &sseServerConfig)
	}

	return sseCmd, sseConfig
}

// newSSEHandler builds the SSE handler for cfg. The HTTP+SSE transport
// predates streamable HTTP and is kept for clients that still speak it.
func newSSEHandler(cfg *config.SSEServerConfig) http.Handler {
	srv := newMCPServer()
	return mcp.NewSSEHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.SSEOptions{DisableLocalhostProtection: cfg.DisableLocalhostProtection},
	)
}

// newSSEMux wires the served handler: the SSE handler, the CORS and origin
// check around it, and the base path route. It also reports the SSE endpoint.
func newSSEMux(cfg *config.SSEServerConfig) (handler http.Handler, ssePath string) {
	basePath := normalizeHTTPPath(cfg.BasePath)
	mux := http.NewServeMux()
	mux.Handle(basePath+"/", corsMiddleware(cfg.AllowedOrigins, newSSEHandler(cfg)))

	return mux, basePath + "/sse"
}

// runSSEServer starts a server that listens for Server-Sent Events (SSE) on the specified address.
func runSSEServer(ctx context.Context, cfg *config.SSEServerConfig) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger, err := initLogger(cfg.LogFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	mux, ssePath := newSSEMux(cfg)

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
