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
	"testing"
)

// The SDK validates inputs against the inferred schema before the handler
// runs; the previous SDK ignored unknown fields, so the schema must not
// reject them at any nesting level.
func TestInferSchemaAllowsUnknownFieldsAtEveryLevel(t *testing.T) {
	schema := InferSchema[LogQueryRequest]()

	if schema.AdditionalProperties != nil {
		t.Fatal("top-level additionalProperties not removed")
	}

	items := schema.Properties["tags"].Items
	if items == nil {
		t.Fatal("tags items schema missing")
	}
	if items.AdditionalProperties != nil {
		t.Fatal("nested tag additionalProperties not removed")
	}
}
