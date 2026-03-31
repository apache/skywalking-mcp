// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tools

import (
	"bytes"
	"io"
	"regexp"

	log "github.com/sirupsen/logrus"
)

// sensitiveFieldPattern matches JSON fields whose values should be redacted in logs.
var sensitiveFieldPattern = regexp.MustCompile(`(?i)("(?:authorization|password|token|secret)"\s*:\s*")((?:[^"\\]|\\.)*)(")`) //nolint:lll // regex must be on one line

// redactSensitiveData masks values of sensitive JSON fields before logging.
func redactSensitiveData(data string) string {
	return sensitiveFieldPattern.ReplaceAllString(data, `${1}[REDACTED]${3}`)
}

// IOLogger is a wrapper around io.Reader and io.Writer that logs complete
// newline-delimited JSON-RPC messages. Partial chunks are held in per-direction
// buffers until a newline arrives, ensuring the redaction regex always sees a
// full message and secrets split across read boundaries are never partially logged.
type IOLogger struct {
	reader   io.Reader
	writer   io.Writer
	logger   *log.Logger
	readBuf  bytes.Buffer
	writeBuf bytes.Buffer
}

// NewIOLogger creates a new IOLogger instance
func NewIOLogger(r io.Reader, w io.Writer, logger *log.Logger) *IOLogger {
	return &IOLogger{
		reader: r,
		writer: w,
		logger: logger,
	}
}

// logCompleteLines drains newline-terminated lines from buf, redacts each one,
// and logs it under the given direction label. Any trailing partial line is left
// in buf for the next call.
func (l *IOLogger) logCompleteLines(buf *bytes.Buffer, direction string) {
	data := buf.Bytes()
	for {
		idx := bytes.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		line := bytes.TrimRight(data[:idx], "\r")
		l.logger.Infof("[%s]: %s", direction, redactSensitiveData(string(line)))
		data = data[idx+1:]
	}
	buf.Reset()
	buf.Write(data)
}

// Read reads data from the underlying io.Reader and logs complete lines.
func (l *IOLogger) Read(p []byte) (n int, err error) {
	if l.reader == nil {
		return 0, io.EOF
	}
	n, err = l.reader.Read(p)
	if n > 0 {
		l.readBuf.Write(p[:n])
		l.logCompleteLines(&l.readBuf, "stdin")
	}
	return n, err
}

// Write writes data to the underlying io.Writer and logs complete lines.
func (l *IOLogger) Write(p []byte) (n int, err error) {
	if l.writer == nil {
		return 0, io.ErrClosedPipe
	}
	l.writeBuf.Write(p)
	l.logCompleteLines(&l.writeBuf, "stdout")
	return l.writer.Write(p)
}
