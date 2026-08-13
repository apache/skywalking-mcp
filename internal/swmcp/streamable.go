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

	"github.com/apache/skywalking-mcp/internal/config"
)

func NewStreamable() *cobra.Command {
	cmd, _ := newStreamableCommand()
	return cmd
}

// newStreamableCommand also returns the accessor for the configuration the
// command runs with, so callers observe exactly what RunE consumes.
func newStreamableCommand() (cmd *cobra.Command, cfg func() config.StreamableServerConfig) {
	streamableCmd := &cobra.Command{
		Use:   "streamable",
		Short: "Start Streamable server",
		Long:  `Starting SkyWalking MCP server with Streamable HTTP transport.`,
	}

	// Add Streamable server specific flags
	v := bindHTTPTransportFlags(streamableCmd)
	streamableCmd.Flags().String("address", "localhost:8000",
		"The host and port to start the Streamable server on")
	streamableCmd.Flags().String("endpoint-path", "/mcp",
		"The path for the streamable-http server")
	_ = v.BindPFlag("address", streamableCmd.Flags().Lookup("address"))
	_ = v.BindPFlag("endpoint-path", streamableCmd.Flags().Lookup("endpoint-path"))

	streamableConfig := func() config.StreamableServerConfig {
		return config.StreamableServerConfig{
			Address:             v.GetString("address"),
			EndpointPath:        v.GetString("endpoint-path"),
			HTTPTransportConfig: httpTransportConfigFrom(v),
		}
	}

	streamableCmd.RunE = func(_ *cobra.Command, _ []string) error {
		if err := validateConfiguredSkyWalkingURL(); err != nil {
			return err
		}

		cfg := streamableConfig()

		return runStreamableServer(&cfg)
	}

	return streamableCmd, streamableConfig
}

// newStreamableHandler builds the streamable HTTP handler for cfg.
func newStreamableHandler(cfg *config.StreamableServerConfig) http.Handler {
	// Stateless is required for protocol version 2026-07-28, which carries
	// identity and capabilities per request instead of in a session.
	srv := newMCPServer()
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{
			Stateless:                  true,
			DisableLocalhostProtection: cfg.DisableLocalhostProtection,
		},
	)
}

// newStreamableMux wires the served handler: the MCP handler, the CORS and
// origin check around it, and the endpoint routes.
func newStreamableMux(cfg *config.StreamableServerConfig) (handler http.Handler, endpoint string) {
	endpointPath := normalizeHTTPPath(cfg.EndpointPath)
	if endpointPath == "" {
		endpointPath = "/"
	}

	wrapped := corsMiddleware(cfg.AllowedOrigins, newStreamableHandler(cfg))
	mux := http.NewServeMux()
	mux.Handle(endpointPath, wrapped)
	if endpointPath != "/" {
		// Accept the trailing-slash form too; the previous server did not
		// route on the path at all.
		mux.Handle(endpointPath+"/", wrapped)
	}

	return mux, endpointPath
}

// runStreamableServer starts the Streamable server with the provided configuration.
func runStreamableServer(cfg *config.StreamableServerConfig) error {
	mux, endpointPath := newStreamableMux(cfg)

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
