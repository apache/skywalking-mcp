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
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"

	"github.com/apache/skywalking-mcp/internal/tools"
)

// newMcpServer creates a new MCP server instance,
// and we can add various tools and capabilities to it.
func newMcpServer() *server.MCPServer {
	mcpServer := server.NewMCPServer(
		"skywalking-mcp",
		"0.1.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging())

	tools.AddTraceTools(mcpServer)

	return mcpServer
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
