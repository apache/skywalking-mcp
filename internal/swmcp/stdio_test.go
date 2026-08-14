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
	"errors"
	"strings"
	"testing"
)

type failingWriter struct {
	fail bool
	out  bytes.Buffer
}

func (f *failingWriter) Write(p []byte) (int, error) {
	if f.fail {
		return 0, errors.New("writer unavailable")
	}
	return f.out.Write(p)
}

// A transient log-writer failure must not replay already-logged lines once
// writes succeed again.
func TestRedactingWriterDoesNotReplayLinesAfterWriteError(t *testing.T) {
	out := &failingWriter{}
	w := &redactingWriter{w: out}

	if _, err := w.Write([]byte("first\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	out.fail = true
	if _, err := w.Write([]byte("second\n")); err == nil {
		t.Fatal("expected error from failing writer")
	}
	out.fail = false
	if _, err := w.Write([]byte("third\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := out.out.String(); got != "first\nthird\n" {
		t.Fatalf("logged lines = %q, want %q", got, "first\nthird\n")
	}
}

func TestRedactingWriterMasksSensitiveFields(t *testing.T) {
	var out bytes.Buffer
	w := &redactingWriter{w: &out}

	// The secret is split across two writes: nothing may reach the underlying
	// writer until the line is complete and redacted.
	if _, err := w.Write([]byte(`read: {"password":"hun`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("partial line leaked to the log: %q", out.String())
	}
	if _, err := w.Write([]byte(`ter2","Authorization":"Bearer s3cr3t","user":"u"}` + "\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := out.String()
	for _, leaked := range []string{"hunter2", "s3cr3t"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("secret %q leaked to the log: %q", leaked, got)
		}
	}
	if !strings.Contains(got, `"password":"[REDACTED]"`) || !strings.Contains(got, `"Authorization":"[REDACTED]"`) {
		t.Fatalf("sensitive fields not redacted: %q", got)
	}
	if !strings.Contains(got, `"user":"u"`) {
		t.Fatalf("non-sensitive field mangled: %q", got)
	}
}
