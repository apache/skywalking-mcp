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

	"github.com/mark3labs/mcp-go/server"
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
	_ = viper.BindPFlag("address", streamableCmd.Flags().Lookup("address"))
	_ = viper.BindPFlag("endpoint-path", streamableCmd.Flags().Lookup("endpoint-path"))

	return streamableCmd
}

// runStreamableServer starts the Streamable server with the provided configuration.
func runStreamableServer(cfg *config.StreamableServerConfig) error {
	httpServer := server.NewStreamableHTTPServer(
		newMcpServer(),
		server.WithStateLess(true),
		server.WithLogger(log.StandardLogger()),
		server.WithHTTPContextFunc(EnhanceHTTPContextFunc()),
		server.WithEndpointPath(viper.GetString("endpoint-path")),
	)
	log.Infof("streamable HTTP server listening on %s%s\n", cfg.Address, cfg.EndpointPath)

	if err := httpServer.Start(cfg.Address); err != nil {
		return fmt.Errorf("streamable HTTP server error: %v", err)
	}

	return nil
}
