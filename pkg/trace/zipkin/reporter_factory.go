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

package zipkin

import (
	"time"

	"github.com/openzipkin/zipkin-go/reporter"
	"github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/openzipkin/zipkin-go/reporter/kafka"
)

var (
	factory = make(map[string]ReporterBuilder)
)

func init() {
	factory[ZipkinHttpReport] = HttpReporterBuilder
	factory[ZipkinKafkaReport] = KafkaReporterBuilder
}

func GetReportBuilder(typ string) (ReporterBuilder, bool) {
	if v, ok := factory[typ]; ok {
		return v, ok
	}
	return nil, false
}

type ReporterBuilder func(ZipkinTraceConfig) (reporter.Reporter, error)

func HttpReporterBuilder(cfg ZipkinTraceConfig) (reporter.Reporter, error) {
	opts := make([]http.ReporterOption, 0, 3)
	if cfg.HttpConfig.Timeout > 0 {
		opts = append(opts, http.Timeout(time.Second*time.Duration(cfg.HttpConfig.Timeout)))

	}
	if cfg.HttpConfig.BatchInterval > 0 {
		opts = append(opts, http.BatchInterval(time.Second*time.Duration(cfg.HttpConfig.BatchInterval)))
	}
	if cfg.HttpConfig.BatchSize > 0 {
		opts = append(opts, http.BatchSize(cfg.HttpConfig.BatchSize))
	}
	return http.NewReporter(cfg.Addresses[0], opts...), nil
}

func KafkaReporterBuilder(cfg ZipkinTraceConfig) (reporter.Reporter, error) {
	opts := make([]kafka.ReporterOption, 0, 1)
	if cfg.KafkaConfig.Topic != "" {
		opts = append(opts, kafka.Topic(cfg.KafkaConfig.Topic))

	}
	return kafka.NewReporter(cfg.Addresses, opts...)
}
