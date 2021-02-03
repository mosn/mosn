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

package sync

// WorkerPool provides a pool for goroutines
type WorkerPool interface {

	// Schedule try to acquire pooled worker goroutine to execute the specified task,
	// this method would block if no worker goroutine is available
	Schedule(task func())

	// Schedule try to acquire pooled worker goroutine to execute the specified task first,
	// but would not block if no worker goroutine is available. A temp goroutine will be created for task execution.
	ScheduleAlways(task func())

	ScheduleAuto(task func())
}
