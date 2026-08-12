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
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/skywalking-mcp/internal/config"
)

// sensitiveFieldPattern matches JSON fields whose values should be redacted in logs.
var sensitiveFieldPattern = regexp.MustCompile(`(?i)("(?:authorization|password|token|secret)"\s*:\s*")((?:[^"\\]|\\.)*)(")`) //nolint:lll // regex must be on one line

// redactingWriter masks values of sensitive JSON fields before forwarding to
// the log writer. Data is buffered per line so the redaction regex always sees
// a full message and secrets split across writes are never partially logged.
// It is safe for concurrent use.
type redactingWriter struct {
	w   io.Writer
	mu  sync.Mutex
	buf bytes.Buffer
}

func (r *redactingWriter) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.buf.Write(p)
	data := r.buf.Bytes()
	var werr error
	for werr == nil {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		line := sensitiveFieldPattern.ReplaceAll(data[:idx+1], []byte(`${1}[REDACTED]${3}`))
		// A failed line is dropped rather than retried: keeping it buffered
		// would rewrite the lines already logged on the next call.
		_, werr = r.w.Write(line)
		data = data[idx+1:]
	}
	rest := append([]byte(nil), data...)
	r.buf.Reset()
	r.buf.Write(rest)
	return len(p), werr
}

func NewStdioServer() *cobra.Command {
	return &cobra.Command{
		Use:   "stdio",
		Short: "Start stdio server",
		Long:  `Start a server that communicates via standard input/output streams using JSON-RPC messages.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateConfiguredSkyWalkingURL(); err != nil {
				return err
			}

			stdioServerConfig := config.StdioServerConfig{
				URL:         viper.GetString("url"),
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

	logger, err := initLogger(cfg.LogFilePath)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	var transport mcp.Transport = &mcp.StdioTransport{}
	if cfg.LogCommands {
		transport = &mcp.LoggingTransport{Transport: transport, Writer: &redactingWriter{w: logger.Writer()}}
	}

	// Start listening for messages
	errC := make(chan error, 1)
	go func() {
		errC <- newMCPServer().Run(ctx, transport)
	}()

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
