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
	"github.com/openzipkin/zipkin-go/model"
	"github.com/openzipkin/zipkin-go/propagation/b3"
)

const (
	traceIDHeader      = "X-B3-Traceid"      // b3.TraceID
	spanIDHeader       = "X-B3-Spanid"       // b3.SpanID
	parentSpanIDHeader = "X-B3-ParentSpanid" // b3.ParentSpanID
	sampledHeader      = "X-B3-Sampled"      // b3.Sampled
	flagsHeader        = "X-B3-Flags"        // b3.Flags
)

func getBoth(g Getter, k1, k2 string) string {
	v := g.Get(k1)
	if v != "" {
		return v
	}
	return g.Get(k2)
}

// ExtractB3 will extract a span.Context from the HTTP Request if found in B3 header format.
func ExtractB3(g Getter) (*model.SpanContext, error) {
	var (
		traceIDHeader      = getBoth(g, b3.TraceID, traceIDHeader)
		spanIDHeader       = getBoth(g, b3.SpanID, spanIDHeader)
		parentSpanIDHeader = getBoth(g, b3.ParentSpanID, parentSpanIDHeader)
		sampledHeader      = getBoth(g, b3.Sampled, sampledHeader)
		flagsHeader        = getBoth(g, b3.Flags, flagsHeader)
	)

	var (
		sc   *model.SpanContext
		sErr error
		mErr error
	)

	sc, mErr = b3.ParseHeaders(
		traceIDHeader, spanIDHeader, parentSpanIDHeader,
		sampledHeader, flagsHeader,
	)

	if mErr != nil && sErr != nil {
		return nil, sErr
	}

	return sc, mErr
}

// InjectB3 will inject a span.Context into a HTTP Request
func InjectB3(s Setter, sc model.SpanContext) error {
	if (model.SpanContext{}) == sc {
		return b3.ErrEmptyContext
	}

	if sc.Debug {
		s.Set(b3.Flags, "1")
	} else if sc.Sampled != nil {
		// Debug is encoded as X-B3-Flags: 1. Since Debug implies Sampled,
		// so don't also send "X-B3-Sampled: 1".
		if *sc.Sampled {
			s.Set(b3.Sampled, "1")
		} else {
			s.Set(b3.Sampled, "0")
		}
	}

	if !sc.TraceID.Empty() && sc.ID > 0 {
		s.Set(b3.TraceID, sc.TraceID.String())
		s.Set(b3.SpanID, sc.ID.String())
		if sc.ParentID != nil {
			s.Set(b3.ParentSpanID, sc.ParentID.String())
		}
	}

	return nil
}

type Getter interface {
	Get(string) string
}

type Setter interface {
	Set(string, string)
}

type Carrier interface {
	Getter
	Setter
	ForeachKey(handler func(key, val string) error) error
}
