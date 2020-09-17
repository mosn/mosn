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

const (
	ZipkinTracer = "Zipkin"
)

type HeaderCarrier struct {
	HeaderMap HeaderMap
}

func (c *HeaderCarrier) Get(key string) string {
	val, _ := c.HeaderMap.Get(key)
	return val
}

// Set conforms to the TextMapWriter interface.
func (c *HeaderCarrier) Set(key, val string) {
	c.HeaderMap.Set(key, val)
}

// ForeachKey conforms to the TextMapReader interface.
func (c *HeaderCarrier) ForeachKey(handler func(key, val string) error) error {
	c.HeaderMap.Range(func(key, value string) bool {
		return handler(key, value) == nil
	})
	return nil
}

type HeaderMap interface {
	Get(key string) (string, bool)
	Set(key, value string)
	Range(f func(key, value string) bool)
}
