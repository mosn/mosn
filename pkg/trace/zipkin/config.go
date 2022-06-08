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
	"errors"
)

const (
	ZipkinKafkaReport string = "kafka"
	ZipkinHttpReport  string = "http"
)

type ZipkinTraceConfig struct {
	ServiceName string             `json:"service_name"`
	Reporter    string             `json:"reporter"`
	SampleRate  float64            `json:"sample_rate"`
	Addresses   []string           `json:"addresses"`
	HttpConfig  *HttpReportConfig  `json:"http"`
	KafkaConfig *KafkaReportConfig `json:"kafka"`
}

type HttpReportConfig struct {
	Timeout       int `json:"timeout"`
	BatchSize     int `json:"batch_size"`
	BatchInterval int `json:"batch_interval"`
}

type KafkaReportConfig struct {
	Topic string `json:"topic"`
}

func (z *ZipkinTraceConfig) ValidateZipkinConfig() (bool, error) {
	if z.SampleRate > 1 || z.SampleRate < 0 {
		return false, errors.New("sample rate should between 1.0 and 0.0")
	}

	switch z.Reporter {
	case ZipkinHttpReport:
		if len(z.Addresses) != 1 {
			return false, errors.New("http config only support single address")
		}
	case ZipkinKafkaReport:
		if len(z.Addresses) <= 0 {
			return false, errors.New("kafka config address can't be empty")
		}
	}
	return true, nil
}
