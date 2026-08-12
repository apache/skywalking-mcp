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
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func ResultText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// ResultError reports an expected query failure to the model instead of
// surfacing it as a protocol error.
func ResultError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// InferSchema derives the JSON schema of a tool's input type. Registration
// happens at startup, so an inference failure is a programming error.
//
// A parameterless input infers to a schema without "properties" at all, which
// OpenAI's strict mode rejects, so the empty object is filled in.
func InferSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("infer schema for %T: %v", *new(T), err))
	}
	if schema.Properties == nil {
		schema.Properties = map[string]*jsonschema.Schema{}
	}
	// Inference marks structs additionalProperties:false and the SDK validates
	// inputs before the handler runs. The previous SDK ignored unknown fields
	// and clients do send extras, so restore the lenient behavior at every
	// nesting level.
	relaxAdditionalProperties(schema)
	return schema
}

// relaxAdditionalProperties removes the additionalProperties:false constraint
// (represented as {"not": {}}) from schema and every nested object schema.
func relaxAdditionalProperties(schema *jsonschema.Schema) {
	if schema == nil {
		return
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Not != nil {
		schema.AdditionalProperties = nil
	}
	for _, property := range schema.Properties {
		relaxAdditionalProperties(property)
	}
	relaxAdditionalProperties(schema.Items)
}

// stepEnum lists the time-step granularities shared by the query tools. The
// MQE tool additionally accepts MONTH.
var stepEnum = []string{"SECOND", "MINUTE", "HOUR", "DAY"}

// WithRequired declares which properties a tool requires. Inference derives
// requiredness from the absence of omitempty, which ties the API contract to
// serialization behavior; declaring it here keeps the contract explicit and
// reviewable.
func WithRequired(schema *jsonschema.Schema, properties ...string) {
	for _, property := range properties {
		if _, ok := schema.Properties[property]; !ok {
			panic(fmt.Sprintf("required on unknown property %q", property))
		}
	}
	schema.Required = properties
}

// WithDescriptions sets property descriptions that a struct tag cannot hold,
// such as ones containing backticks.
func WithDescriptions(schema *jsonschema.Schema, descriptions map[string]string) {
	for property, description := range descriptions {
		prop, ok := schema.Properties[property]
		if !ok {
			panic(fmt.Sprintf("description on unknown property %q", property))
		}
		prop.Description = description
	}
}

// WithEnum constrains a property to a fixed set of values. The jsonschema
// struct tag only carries a description, so enums are applied to the inferred
// schema instead.
func WithEnum(schema *jsonschema.Schema, property string, values ...string) {
	prop, ok := schema.Properties[property]
	if !ok {
		panic(fmt.Sprintf("enum on unknown property %q", property))
	}
	prop.Enum = make([]any, len(values))
	for i, v := range values {
		prop.Enum[i] = v
	}
}
