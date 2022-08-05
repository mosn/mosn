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
	"time"
)

// https://istio.io/latest/docs/reference/config/proxy_extensions/stats/#MetricDefinition
type StatsConfig struct {
	// The following settings should be rarely used. Enable debug for this filter.
	Debug string `json:"debug"`
	// maximum size of the peer metadata cache. A long lived proxy that connects with many transient peers can build up a large cache. To turn off the cache, set this field to a negative value.
	MaxPeerCacheSize int `json:"max_peer_cache_size"`
	// prefix to add to stats emitted by the plugin.
	StatPrefix string `json:"stat_prefix"`
	// Stats api squashes dimensions in a single string. The squashed string is parsed at prometheus scrape time to recover dimensions.
	// The following 2 fields set the field and value separators {key: value} –> key{valueseparator}value{fieldseparator}
	FieldSeparator string `json:"field_separator"`
	// default: “==”
	ValueSeparator string `json:"value_separator"`
	// Disable using host header as a fallback if destination service is not available from the controlplane. Disable the fallback if the host header originates outsides the mesh, like at ingress.
	DisableHostHeaderFallback bool `json:"disable_host_header_fallback,omitempty"`
	// Allows configuration of the time between calls out to for TCP metrics reporting. The default duration is 15s.
	TcpReportingDuration time.Duration `json:"tcp_reporting_duration,omitempty"`
	// Metric overrides.
	Metrics []*MetricConfig `json:"metrics"`
	// Metric definitions.
	Definitions []*MetricDefinition `json:"definitions"`
}

type MetricConfig struct {
	// Metric name to restrict the override to a metric. If not specified, applies to all.
	Name string `json:"name,omitempty"`
	// Collection of tag names and tag expressions to include in the metric.
	// Conflicts are resolved by the tag name by overriding previously supplied values.
	Dimensions map[string]string `json:"dimensions,omitempty"`
	// A list of tags to remove.
	TagsToRemove []string `json:"tags_to_remove,omitempty"`
	// Conditional enabling the override.
	Match string `json:"match,omitempty"`
}

type MetricDefinition struct {
	// Metric name
	Name string `json:"name,omitempty"`
	// Metric value expression.
	Value string `json:"value,omitempty"`
	// Metric type.
	Type MetricType `json:"type,omitempty"`
}

type MetricType string

const (
	MetricTypeCounter   MetricType = "COUNTER"
	MetricTypeGauge     MetricType = "GAUGE"
	MetricTypeHistogram MetricType = "HISTOGRAM"
)

// https://github.com/istio/proxy/pull/2414/files#diff-db83bcb3df7f25cfb88ff0a20bbd5540R82

var defaultMetricConfig = []*MetricConfig{
	{
		Name:       "",
		Dimensions: defaultDimensions,
	},
}
var defaultMetricDefinition = []*MetricDefinition{
	{
		Name:  "requests_total",
		Value: `request.total_size | 0`,
		Type:  MetricTypeCounter,
	},
	{
		Name:  "request_duration_milliseconds",
		Value: `response.duration | "0"`,
		Type:  MetricTypeHistogram,
	},
	{
		Name:  "request_bytes",
		Value: `request.size | 0`,
		Type:  MetricTypeHistogram,
	},
	{
		Name:  "response_bytes",
		Value: `response.size | 0`,
		Type:  MetricTypeHistogram,
	},
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
