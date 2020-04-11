/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package skywalking

import (
	"context"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func TestNewGO2SkyTracer(t *testing.T) {
	// use default config
	tracer, err := newGO2SkyTracer(nil)
	if err != nil {
		t.Errorf("new go2sky.Tracer error, err: %v", err)
	}
	s, _, err := tracer.CreateEntrySpan(context.Background(), "/NewGO2SkyTracer", func() (s string, err error) {
		return "", nil
	})
	if err != nil {
		t.Errorf("create entry span error, err: %v", err)
	}

	s.End()
	time.Sleep(time.Second)
}

func TestParseAndVerifySkyTracerConfig(t *testing.T) {
	// ReporterCfgErr
	cfg := make(map[string]interface{})
	cfg["reporter"] = "mosn"
	_, err := parseAndVerifySkyTracerConfig(cfg)
	if err == nil || err != ReporterCfgErr {
		t.Errorf("did not verify the reporter config correctly")
	}

	// BackendServiceCfgErr
	cfg = make(map[string]interface{})
	cfg["reporter"] = v2.GRPCReporter
	_, err = parseAndVerifySkyTracerConfig(cfg)
	if err == nil || err != BackendServiceCfgErr {
		t.Errorf("did not verify the  backend service config correctly")
	}
}
