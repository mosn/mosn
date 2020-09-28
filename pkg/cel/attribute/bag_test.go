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

package attribute

import (
	"reflect"
	"testing"
)

func TestMutableBagForMap(t *testing.T) {
	bag := NewMutableBagForMap(map[string]interface{}{
		"base": "base",
	})

	bag = NewMutableBag(bag)
	bag.Set("key", "value")
	v, ok := bag.Get("key")
	if !ok || !reflect.DeepEqual(v, "value") {
		t.Fail()
	}

	v, ok = bag.Get("key2")
	if ok || v != nil {
		t.Fail()
	}

	bag.Set("key2", "value2")
	v, ok = bag.Get("key2")
	if !ok || !reflect.DeepEqual(v, "value2") {
		t.Fail()
	}

	bag.Delete("key")

	v, ok = bag.Get("key")
	if ok || v != nil {
		t.Fail()
	}

	bag.Reset()
	v, ok = bag.Get("key2")
	if ok || v != nil {
		t.Fail()
	}

	v, ok = bag.Get("base")
	if !ok || !reflect.DeepEqual(v, "base") {
		t.Fail()
	}
}
