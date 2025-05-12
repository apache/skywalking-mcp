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

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Tool[T any, R any] struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, args T) (R, error)
	Options     []mcp.ToolOption
}

func NewTool[T any, R any](
	name, desc string,
	handler func(ctx context.Context, args T) (R, error),
	options ...mcp.ToolOption,
) *Tool[T, R] {
	return &Tool[T, R]{
		Name:        name,
		Description: desc,
		Handler:     handler,
		Options:     options,
	}
}

// Register registers the tool with the given MCP server.
func (t *Tool[T, R]) Register(server *server.MCPServer) {
	tool, handler, err := ConvertTool[T, R](t.Name, t.Description, t.Handler, t.Options...)
	if err != nil {
		panic(err)
	}

	server.AddTool(tool, handler)
}

func ConvertTool[T any, R any](
	name string,
	desc string,
	handlerFunc func(ctx context.Context, args T) (R, error),
	options ...mcp.ToolOption,
) (mcp.Tool, server.ToolHandlerFunc, error) {
	baseOptions := []mcp.ToolOption{
		mcp.WithDescription(desc),
		mcp.WithTitleAnnotation(name),
		mcp.WithIdempotentHintAnnotation(true), // we assume tools are idempotent by default
	}

	baseOptions = append(baseOptions, options...)
	tool := mcp.NewTool(name, baseOptions...)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args T
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		result, err := handlerFunc(ctx, args)
		if err != nil {
			return nil, err
		}

		switch v := any(result).(type) {
		case *mcp.CallToolResult:
			return v, nil
		case mcp.CallToolResult:
			return &v, nil
		case nil:
			return nil, nil
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal return value: %s", err)
			}
			return mcp.NewToolResultText(string(jsonBytes)), nil
		}
	}

	return tool, handler, nil
}
