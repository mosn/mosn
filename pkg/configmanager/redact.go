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

package configmanager

import (
	v2 "mosn.io/mosn/pkg/config/v2"
)

// redactedCopy returns a deep copy of the effectiveConfig with all inline TLS
// private keys redacted. It is the single source of the redacted snapshot used
// by the admin config_dump API, so that secret material never leaks through a
// dump response.
func redactedCopy(src effectiveConfig) effectiveConfig {
	dst := src
	dst.MosnConfig = redactedMosnConfig(src.MosnConfig)
	dst.Listener = redactedListeners(src.Listener)
	dst.Cluster = redactedClusters(src.Cluster)
	// Routers / ExtendConfigs do not carry inline TLS private keys.
	return dst
}

func redactedMosnConfig(src v2.MOSNConfig) v2.MOSNConfig {
	dst := src
	// cluster manager TLS context
	redactTLSConfig(&dst.ClusterManager.TLSContext)
	// listener filter chains
	for i := range dst.Servers {
		for j := range dst.Servers[i].Listeners {
			redactListener(&dst.Servers[i].Listeners[j])
		}
	}
	return dst
}

func redactedListeners(src map[string]v2.Listener) map[string]v2.Listener {
	if src == nil {
		return nil
	}
	dst := make(map[string]v2.Listener, len(src))
	for k, l := range src {
		redactListener(&l)
		dst[k] = l
	}
	return dst
}

func redactListener(l *v2.Listener) {
	// FilterChains is a slice; copy the backing array so the redaction never
	// mutates the source config (slices share element storage across copies).
	chains := make([]v2.FilterChain, len(l.FilterChains))
	copy(chains, l.FilterChains)
	l.FilterChains = chains
	for i := range l.FilterChains {
		fc := &l.FilterChains[i]
		// FilterChain.MarshalJSON copies TLSContexts into TLSConfigs (and clears
		// TLSConfig) when TLSContexts is non-empty, so all three fields must be
		// redacted to guarantee the placeholder reaches the dumped output. Each
		// is re-sliced so storage is independent of the source.
		if len(fc.TLSContexts) > 0 {
			ctxs := make([]v2.TLSConfig, len(fc.TLSContexts))
			copy(ctxs, fc.TLSContexts)
			fc.TLSContexts = ctxs
			for j := range fc.TLSContexts {
				redactTLSConfig(&fc.TLSContexts[j])
			}
		}
		if fc.TLSConfig != nil {
			tls := *fc.TLSConfig
			redactTLSConfig(&tls)
			fc.TLSConfig = &tls
		}
		if len(fc.TLSConfigs) > 0 {
			cfgs := make([]v2.TLSConfig, len(fc.TLSConfigs))
			copy(cfgs, fc.TLSConfigs)
			fc.TLSConfigs = cfgs
			for j := range fc.TLSConfigs {
				redactTLSConfig(&fc.TLSConfigs[j])
			}
		}
	}
}

func redactedClusters(src map[string]v2.Cluster) map[string]v2.Cluster {
	if src == nil {
		return nil
	}
	dst := make(map[string]v2.Cluster, len(src))
	for k, c := range src {
		redactTLSConfig(&c.TLS)
		dst[k] = c
	}
	return dst
}
