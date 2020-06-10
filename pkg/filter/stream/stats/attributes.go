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

package stats

import (
	"sort"

	"mosn.io/mosn/pkg/cel/attribute"
)

type StatsConfig struct {
	Debug      string         `json:"debug,omitempty"`
	StatPrefix string         `json:"stat_prefix,omitempty"`
	Metrics    []MetricConfig `json:"metrics,omitempty"`
}

type MetricConfig struct {
	Name       string            `json:"name,omitempty"`
	Dimensions map[string]string `json:"dimensions,omitempty"`
	Value      string            `json:"value,omitempty"`
}

var attributemanifest = map[string]attribute.Kind{
	// istio-proxy
	"origin.ip":                        attribute.IP_ADDRESS,
	"origin.uid":                       attribute.STRING,
	"origin.user":                      attribute.STRING,
	"request.headers":                  attribute.STRING_MAP,
	"request.id":                       attribute.STRING,
	"request.host":                     attribute.STRING,
	"request.method":                   attribute.STRING,
	"request.path":                     attribute.STRING,
	"request.url_path":                 attribute.STRING,
	"request.query_params":             attribute.STRING_MAP,
	"request.reason":                   attribute.STRING,
	"request.referer":                  attribute.STRING,
	"request.scheme":                   attribute.STRING,
	"request.total_size":               attribute.INT64,
	"request.size":                     attribute.INT64,
	"request.time":                     attribute.TIMESTAMP,
	"request.useragent":                attribute.STRING,
	"response.code":                    attribute.INT64,
	"response.duration":                attribute.DURATION,
	"response.headers":                 attribute.STRING_MAP,
	"response.total_size":              attribute.INT64,
	"response.size":                    attribute.INT64,
	"response.time":                    attribute.TIMESTAMP,
	"response.grpc_status":             attribute.STRING,
	"response.grpc_message":            attribute.STRING,
	"source.uid":                       attribute.STRING,
	"source.user":                      attribute.STRING, // DEPRECATED
	"source.principal":                 attribute.STRING,
	"destination.uid":                  attribute.STRING,
	"destination.principal":            attribute.STRING,
	"destination.port":                 attribute.INT64,
	"connection.event":                 attribute.STRING,
	"connection.id":                    attribute.STRING,
	"connection.received.bytes":        attribute.INT64,
	"connection.received.bytes_total":  attribute.INT64,
	"connection.sent.bytes":            attribute.INT64,
	"connection.sent.bytes_total":      attribute.INT64,
	"connection.duration":              attribute.DURATION,
	"connection.mtls":                  attribute.BOOL,
	"connection.requested_server_name": attribute.STRING,
	"context.protocol":                 attribute.STRING,
	"context.proxy_error_code":         attribute.STRING,
	"context.timestamp":                attribute.TIMESTAMP,
	"context.time":                     attribute.TIMESTAMP,

	// Deprecated, kept for compatibility
	"context.reporter.local":              attribute.BOOL,
	"context.reporter.kind":               attribute.STRING,
	"context.reporter.uid":                attribute.STRING,
	"context.proxy_version":               attribute.STRING,
	"api.service":                         attribute.STRING,
	"api.version":                         attribute.STRING,
	"api.operation":                       attribute.STRING,
	"api.protocol":                        attribute.STRING,
	"request.auth.principal":              attribute.STRING,
	"request.auth.audiences":              attribute.STRING,
	"request.auth.presenter":              attribute.STRING,
	"request.auth.claims":                 attribute.STRING_MAP,
	"request.auth.raw_claims":             attribute.STRING,
	"request.api_key":                     attribute.STRING,
	"rbac.permissive.response_code":       attribute.STRING,
	"rbac.permissive.effective_policy_id": attribute.STRING,
	"check.error_code":                    attribute.INT64,
	"check.error_message":                 attribute.STRING,
	"check.cache_hit":                     attribute.BOOL,
	"quota.cache_hit":                     attribute.BOOL,

	// kubernetes
	"source.ip":                      attribute.IP_ADDRESS,
	"source.labels":                  attribute.STRING_MAP,
	"source.metadata":                attribute.STRING_MAP,
	"source.name":                    attribute.STRING,
	"source.namespace":               attribute.STRING,
	"source.owner":                   attribute.STRING,
	"source.serviceAccount":          attribute.STRING,
	"source.services":                attribute.STRING,
	"source.workload.uid":            attribute.STRING,
	"source.workload.name":           attribute.STRING,
	"source.workload.namespace":      attribute.STRING,
	"destination.ip":                 attribute.IP_ADDRESS,
	"destination.labels":             attribute.STRING_MAP,
	"destination.metadata":           attribute.STRING_MAP,
	"destination.owner":              attribute.STRING,
	"destination.name":               attribute.STRING,
	"destination.container.name":     attribute.STRING,
	"destination.namespace":          attribute.STRING,
	"destination.service.uid":        attribute.STRING,
	"destination.service.name":       attribute.STRING,
	"destination.service.namespace":  attribute.STRING,
	"destination.service.host":       attribute.STRING,
	"destination.serviceAccount":     attribute.STRING,
	"destination.workload.uid":       attribute.STRING,
	"destination.workload.name":      attribute.STRING,
	"destination.workload.namespace": attribute.STRING,
}

var defaultMetricConfig []MetricConfig

func init() {
	keys := make([]string, 0, len(defaultMetrics))
	for key := range defaultMetrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := defaultMetrics[key]
		defaultMetricConfig = append(defaultMetricConfig, MetricConfig{
			Name:       key,
			Value:      value,
			Dimensions: defaultDimensions,
		})
	}
}

// https://github.com/istio/proxy/pull/2414/files#diff-db83bcb3df7f25cfb88ff0a20bbd5540R82
var defaultMetrics = map[string]string{
	"requests_total":                `request.total_size | 0`,
	"request_duration_milliseconds": `response.duration | "0"`,
	"request_bytes":                 `request.size | 0`,
	"response_bytes":                `response.size | 0`,
}

var defaultDimensions = map[string]string{
	"reporter":                       `conditional((context.reporter.kind | "inbound") == "outbound", "source", "destination")`,
	"source_workload":                `source.workload.name | "unknown"`,
	"source_workload_namespace":      `source.workload.namespace | "unknown"`,
	"source_principal":               `source.principal | "unknown"`,
	"source_app":                     `source.labels["app"] | "unknown"`,
	"source_version":                 `source.labels["version"] | "unknown"`,
	"destination_workload":           `destination.workload.name | "unknown"`,
	"destination_workload_namespace": `destination.workload.namespace | "unknown"`,
	"destination_principal":          `destination.principal | "unknown"`,
	"destination_app":                `destination.labels["app"] | "unknown"`,
	"destination_version":            `destination.labels["version"] | "unknown"`,
	"destination_service":            `destination.service.host | "unknown"`,
	"destination_service_name":       `destination.service.name | "unknown"`,
	"destination_service_namespace":  `destination.service.namespace | "unknown"`,
	"request_protocol":               `api.protocol | context.protocol | "unknown"`,
	"response_code":                  `response.code | 200`,
	"connection_security_policy":     `conditional((context.reporter.kind | "inbound") == "outbound", "unknown", conditional(connection.mtls | false, "mutual_tls", "none"))`,
	"response_flags":                 `context.proxy_error_code | "-"`,
}
