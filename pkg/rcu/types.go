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

package rcu

import (
	"errors"
	"sync/atomic"
)

// rcu errors
var (
	Timeout = errors.New("update timeout")
	Block   = errors.New("update is running, try again")
)

type element struct {
	i     interface{}
	count int32
}

// Value is an rcu value used as rcu lock, have no export fields, can keep any data
type Value struct {
	element atomic.Value
	expired atomic.Value
	running int32
}
