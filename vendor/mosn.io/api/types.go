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

package api

import (
	"fmt"
	"strings"
	"time"
)

// Metadata field can be used to provide additional information about the route.
// It can be used for configuration, stats, and logging.
// The metadata should go under the filter namespace that will need it.
type Metadata map[string]string

// DurationConfig ia a wrapper for time.Duration, so time config can be written in '300ms' or '1h' format
type DurationConfig struct {
	time.Duration
}

// UnmarshalJSON get DurationConfig.Duration from json file
func (d *DurationConfig) UnmarshalJSON(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

// MarshalJSON
func (d DurationConfig) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}
