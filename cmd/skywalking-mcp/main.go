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

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/swmcp"
)

var (
	version = "version"
	commit  = "commit"
	date    = "date"

	rootCmd = &cobra.Command{
		Use:     "server",
		Short:   "Apache SkyWalking MCP Server.",
		Long:    `This is a server that implements the MCP protocol for Apache SkyWalking.`,
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
	}
)

func init() {
	// Set the environment variable prefix
	viper.SetEnvPrefix("SW")
	// Enable environment variable reading
	viper.AutomaticEnv()
	// All fields with . or - will be replaced with _ for ENV vars
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	rootCmd.SetVersionTemplate("{{.Short}}\n{{.Version}}\n")

	// Add global Flags
	rootCmd.PersistentFlags().String("sw-url", "", "Specify the OAP URL to connect to (e.g. http://localhost:12800)")
	rootCmd.PersistentFlags().String("sse-addr", "localhost:8000", "Which address to listen on for SSE transport")
	rootCmd.PersistentFlags().String("log-level", "info", "Logging level (debug, info, warn, error)")
	rootCmd.PersistentFlags().Bool("read-only", false, "Restrict the server to read-only operations")
	rootCmd.PersistentFlags().Bool("log-command", false, "When true, log commands to the log file")
	rootCmd.PersistentFlags().String("log-file", "", "Path to log file")

	// Bind flag to viper
	_ = viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("sw-url"))
	_ = viper.BindPFlag("sse-addr", rootCmd.PersistentFlags().Lookup("sse-addr"))
	_ = viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
	_ = viper.BindPFlag("read-only", rootCmd.PersistentFlags().Lookup("read-only"))
	_ = viper.BindPFlag("log-command", rootCmd.PersistentFlags().Lookup("log-command"))
	_ = viper.BindPFlag("log-file", rootCmd.PersistentFlags().Lookup("log-file"))

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(viper.GetString("log-level")),
	})))

	// Add subcommands
	rootCmd.AddCommand(swmcp.NewStdioServer())
	rootCmd.AddCommand(swmcp.NewSSEServer())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func parseLogLevel(level string) slog.Level {
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(level)); err != nil {
		return slog.LevelInfo
	}
	return slogLevel
}
