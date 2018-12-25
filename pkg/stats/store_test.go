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

package stats

import (
	"testing"
)

func TestGetAll(t *testing.T) {
	ResetAll()

	// new some stats
	NewStats("type1", map[string]string{"lk": "lv"})
	NewStats("type2", map[string]string{"lk": "lv"})

	if len(GetAll()) != 2 {
		t.Errorf("get all lentgh error, expected 2, actual %d", len(GetAll()))
	}
}
