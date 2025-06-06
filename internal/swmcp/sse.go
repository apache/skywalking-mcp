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

	"github.com/mark3labs/mcp-go/server"
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
	_ = viper.BindPFlag("sse-address", sseCmd.Flags().Lookup("sse-address"))
	_ = viper.BindPFlag("base-path", sseCmd.Flags().Lookup("base-path"))

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

	sseServer := server.NewSSEServer(
		newMcpServer(),
		server.WithStaticBasePath(cfg.BasePath),
		server.WithSSEContextFunc(EnhanceHTTPContextFunc()),
	)
	ssePath := sseServer.CompleteSsePath()
	log.Printf("Starting SkyWalking MCP server using SSE transport listening on http://%s%s\n ", cfg.Address, ssePath)

	errCh := make(chan error, 1)
	go func() {
		if err := sseServer.Start(cfg.Address); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	// Graceful shutdown
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First try to shut down the SSE server
	if err := sseServer.Shutdown(shCtx); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Errorf("Error shutting down SSE server: %v", err)
		}
	}

	// Wait for any remaining operations to complete
	select {
	case <-shCtx.Done():
		return fmt.Errorf("shutdown timed out")
	case <-time.After(100 * time.Millisecond):
		// Give a small grace period for cleanup
	}

	_, _ = fmt.Fprintln(os.Stderr, "SSE server stopped gracefully")
	return nil
}
