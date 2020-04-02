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

package utils

import (
	"fmt"
	"os"
	"runtime/debug"
)

var debugIgnoreStdout = false

// GoWithRecover wraps a `go func()` with recover()
func GoWithRecover(handler func(), recoverHandler func(r interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// TODO: log
				if !debugIgnoreStdout {
					fmt.Fprintf(os.Stderr, "%s goroutine panic: %v\n%s\n", CacheTime(), r, string(debug.Stack()))
				}
				if recoverHandler != nil {
					go func() {
						defer func() {
							if p := recover(); p != nil {
								if !debugIgnoreStdout {
									fmt.Fprintf(os.Stderr, "recover goroutine panic:%v\n%s\n", p, string(debug.Stack()))
								}
							}
						}()
						recoverHandler(r)
					}()
				}
			}
		}()
		handler()
	}()
}
