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

package gxsort

import (
	"sort"
)

func Int64(slice []int64) {
	sort.Sort(Int64Slice(slice))
}

func Int32(slice []int32) {
	sort.Sort(Int32Slice(slice))
}

func Uint32(slice []uint32) {
	sort.Sort(Uint32Slice(slice))
}

type Int64Slice []int64

func (p Int64Slice) Len() int {
	return len(p)
}

func (p Int64Slice) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p Int64Slice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type Int32Slice []int32

func (p Int32Slice) Len() int {
	return len(p)
}

func (p Int32Slice) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p Int32Slice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type Uint32Slice []uint32

func (p Uint32Slice) Len() int {
	return len(p)
}

func (p Uint32Slice) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p Uint32Slice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
