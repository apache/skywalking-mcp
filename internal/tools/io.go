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

// IOLogger is a wrapper around io.Reader and io.Writer that can be used
// to log the data being read and written from the underlying streams
type IOLogger struct {
	reader io.Reader
	writer io.Writer
	logger *log.Logger
}

// NewIOLogger creates a new IOLogger instance
func NewIOLogger(r io.Reader, w io.Writer, logger *log.Logger) *IOLogger {
	return &IOLogger{
		reader: r,
		writer: w,
		logger: logger,
	}
}

// Read reads data from the underlying io.Reader and logs it.
func (l *IOLogger) Read(p []byte) (n int, err error) {
	if l.reader == nil {
		return 0, io.EOF
	}
	n, err = l.reader.Read(p)
	if n > 0 {
		l.logger.Infof("[stdin]: received %d bytes: %s", n, redactSensitiveData(string(p[:n])))
	}
	return n, err
}

// Write writes data to the underlying io.Writer and logs it.
func (l *IOLogger) Write(p []byte) (n int, err error) {
	if l.writer == nil {
		return 0, io.ErrClosedPipe
	}
	l.logger.Infof("[stdout]: sending %d bytes: %s", len(p), redactSensitiveData(string(p)))
	return l.writer.Write(p)
}
