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

// Bag is a generic mechanism to access a set of attributes.
type Bag interface {
	// Get returns an attribute value.
	Get(name string) (value interface{}, found bool)
}

type emptyBag struct{}

var EmptyBag emptyBag

func (emptyBag) Get(name string) (interface{}, bool) { return nil, false }

// MutableBag is a generic mechanism to read and write a set of attributes.
type MutableBag struct {
	parent Bag
	values map[string]interface{}
}

// NewMutableBag returns an initialized bag.
func NewMutableBag(parent Bag) *MutableBag {
	mb := &MutableBag{
		values: map[string]interface{}{},
	}

	if parent == nil {
		mb.parent = EmptyBag
	} else {
		mb.parent = parent
	}

	return mb
}

// NewMutableBagForMap returns a Mutable bag based on the specified map.
func NewMutableBagForMap(values map[string]interface{}) *MutableBag {
	mb := &MutableBag{
		values: values,
		parent: EmptyBag,
	}

	return mb
}

// Get returns an attribute value.
func (mb *MutableBag) Get(name string) (interface{}, bool) {
	r, b := mb.values[name]
	if !b && mb.parent != nil {
		r, b = mb.parent.Get(name)
	}

	return r, b
}

// Set creates an override for a named attribute.
func (mb *MutableBag) Set(name string, value interface{}) {
	mb.values[name] = value
}

// Delete removes a named item from the local state.
// The item may still be present higher in the hierarchy
func (mb *MutableBag) Delete(name string) {
	delete(mb.values, name)
}

// Reset removes all local state.
func (mb *MutableBag) Reset() {
	mb.values = map[string]interface{}{}
}
