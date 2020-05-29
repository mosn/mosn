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

package util

import (
	"fmt"
	"strconv"
)

func DivideInt64(numerator int64, denominator int64) float64 {
	a := float64(numerator)
	b := float64(denominator)
	return DivideFloat64(a, b)
}

func DivideFloat64(numerator float64, denominator float64) float64 {
	value := numerator / denominator
	result, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", value), 64)
	return result
}
