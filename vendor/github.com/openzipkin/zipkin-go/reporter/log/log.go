// Copyright 2019 The OpenZipkin Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
Package log implements a reporter to send spans in V2 JSON format to the Go
standard Logger.
*/
package log

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/reporter"
)

// logReporter will send spans to the default Go Logger.
type logReporter struct {
	logger *log.Logger
}

// NewReporter returns a new log reporter.
func NewReporter(l *log.Logger) reporter.Reporter {
	if l == nil {
		// use standard type of log setup
		l = log.New(os.Stderr, "", log.LstdFlags)
	}
	return &logReporter{
		logger: l,
	}
}

// Send outputs a span to the Go logger.
func (r *logReporter) Send(s model.SpanModel) {
	if b, err := json.MarshalIndent(s, "", "  "); err == nil {
		r.logger.Printf("%s:\n%s\n\n", time.Now(), string(b))
	}
}

// Close closes the reporter
func (*logReporter) Close() error { return nil }
