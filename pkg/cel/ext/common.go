/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0 * (the "License"); you may not use this file except in compliance with
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

package ext

import (
	"fmt"
	"os"
	"time"
)

func init() {
	// get current timestamp
	check(RegisterSimpleFunctionBoth("get_current_timestamp", getCurrMiniTimestamp))
	// get value by environment variable
	check(RegisterSimpleFunctionBoth("get_value_from_env", getValueFromEnv))
}

func getCurrMiniTimestamp() string {
	return  fmt.Sprintf("%.f", float64(time.Now().UnixNano()/ int64(time.Millisecond)))
}

func getValueFromEnv(key string) string {
	return os.Getenv(key)
}
