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
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/config"
)

func NewStreamable() *cobra.Command {
	streamableCmd := &cobra.Command{
		Use:   "streamable",
		Short: "Start Streamable server",
		Long:  `Starting SkyWalking MCP server with Streamable HTTP transport.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateConfiguredSkyWalkingURL(); err != nil {
				return err
			}

			streamableConfig := config.StreamableServerConfig{
				Address:      viper.GetString("address"),
				EndpointPath: viper.GetString("endpoint-path"),
			}

			return runStreamableServer(&streamableConfig)
		},
	}

	// Add Streamable server specific flags
	streamableCmd.Flags().String("address", "localhost:8000",
		"The host and port to start the Streamable server on")
	streamableCmd.Flags().String("endpoint-path", "/mcp",
		"The path for the streamable-http server")
	streamableCmd.Flags().String("allowed-origins", "",
		"Comma-separated allowed CORS origins. Empty = open (any origin reflected). Use * for wildcard header.")
	_ = viper.BindPFlag("address", streamableCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("endpoint-path", streamableCmd.Flags().Lookup("endpoint-path"))
	_ = viper.BindPFlag("allowed-origins", streamableCmd.Flags().Lookup("allowed-origins"))

	return streamableCmd
}

// runStreamableServer starts the Streamable server with the provided configuration.
func runStreamableServer(cfg *config.StreamableServerConfig) error {
	allowedOrigins := parseAllowedOrigins(viper.GetString("allowed-origins"))

	// Stateless is required for protocol version 2026-07-28, which carries
	// identity and capabilities per request instead of in a session.
	srv := newMCPServer()
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	endpointPath := normalizeHTTPPath(cfg.EndpointPath)
	if endpointPath == "" {
		endpointPath = "/"
	}
	wrapped := corsMiddleware(allowedOrigins, handler)
	mux := http.NewServeMux()
	mux.Handle(endpointPath, wrapped)
	if endpointPath != "/" {
		// Accept the trailing-slash form too; the previous server did not
		// route on the path at all.
		mux.Handle(endpointPath+"/", wrapped)
	}

	httpServer := &http.Server{
		Addr:              cfg.Address,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Infof("streamable HTTP server listening on %s%s\n", cfg.Address, endpointPath)

	if err := httpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("streamable HTTP server error: %v", err)
	}

	return nil
}
