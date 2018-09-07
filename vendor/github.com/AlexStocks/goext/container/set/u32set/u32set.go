// Copyright (C) 2017 ScyllaDB
// Use of this source code is governed by a ALv2-style
// license that can be found at https://github.com/scylladb/go-set/LICENSE.

package u32set

import (
	"fmt"
	"strings"
)

var (
	// helpful to not write everywhere struct{}{}
	keyExists   = struct{}{}
	nonExistent uint32
)

// Set is the main set structure that holds all the data
// and methods used to working with the set.
type Set struct {
	m map[uint32]struct{}
}

// New creates and initializes a new Set interface.
func New(ts ...uint32) *Set {
	s := &Set{}
	s.m = make(map[uint32]struct{})
	s.Add(ts...)
	return s
}

// Add includes the specified items (one or more) to the Set. The underlying
// Set s is modified. If passed nothing it silently returns.
func (s *Set) Add(items ...uint32) {
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		s.m[item] = keyExists
	}
}

// Remove deletes the specified items from the Set.  The underlying Set s is
// modified. If passed nothing it silently returns.
func (s *Set) Remove(items ...uint32) {
	if len(items) == 0 {
		return
	}

	for _, item := range items {
		delete(s.m, item)
	}
}

// Pop  deletes and return an item from the Set. The underlying Set s is
// modified. If Set is empty, nil is returned.
func (s *Set) Pop() uint32 {
	for item := range s.m {
		delete(s.m, item)
		return item
	}
	return nonExistent
}

// Has looks for the existence of items passed. It returns false if nothing is
// passed. For multiple items it returns true only if all of  the items exist.
func (s *Set) Has(items ...uint32) bool {
	// assume checked for empty item, which not exist
	if len(items) == 0 {
		return false
	}

	has := true
	for _, item := range items {
		if _, has = s.m[item]; !has {
			break
		}
	}
	return has
}

// Size returns the number of items in a Set.
func (s *Set) Size() int {
	return len(s.m)
}

// Clear removes all items from the Set.
func (s *Set) Clear() {
	s.m = make(map[uint32]struct{})
}

// IsEmpty reports whether the Set is empty.
func (s *Set) IsEmpty() bool {
	return s.Size() == 0
}

// IsEqual test whether s and t are the same in size and have the same items.
func (s *Set) IsEqual(t *Set) bool {
	// return false if they are no the same size
	if sameSize := len(s.m) == t.Size(); !sameSize {
		return false
	}

	equal := true
	t.Each(func(item uint32) bool {
		_, equal = s.m[item]
		return equal // if false, Each() will end
	})

	return equal
}

// IsSubset tests whether t is a subset of s.
func (s *Set) IsSubset(t *Set) bool {
	if len(t.m) > len(s.m) {
		return false
	}

	subset := true

	t.Each(func(item uint32) bool {
		_, subset = s.m[item]
		return subset
	})

	return subset
}

// IsSuperset tests whether t is a superset of s.
func (s *Set) IsSuperset(t *Set) bool {
	return t.IsSubset(s)
}

// Each traverses the items in the Set, calling the provided function for each
// Set member. Traversal will continue until all items in the Set have been
// visited, or if the closure returns false.
func (s *Set) Each(f func(item uint32) bool) {
	for item := range s.m {
		if !f(item) {
			break
		}
	}
}

// Copy returns a new Set with a copy of s.
func (s *Set) Copy() *Set {
	u := New()
	for item := range s.m {
		u.Add(item)
	}
	return u
}

// String returns a string representation of s
func (s *Set) String() string {
	t := make([]string, 0, len(s.List()))
	for _, item := range s.List() {
		t = append(t, fmt.Sprintf("%v", item))
	}

	return fmt.Sprintf("[%s]", strings.Join(t, ", "))
}

// List returns a slice of all items. There is also StringSlice() and
// IntSlice() methods for returning slices of type string or int.
func (s *Set) List() []uint32 {
	list := make([]uint32, 0, len(s.m))

	for item := range s.m {
		list = append(list, item)
	}

	return list
}

// Merge is like Union, however it modifies the current Set it's applied on
// with the given t Set.
func (s *Set) Merge(t *Set) {
	t.Each(func(item uint32) bool {
		s.m[item] = keyExists
		return true
	})
}

// Separate removes the Set items containing in t from Set s. Please aware that
// it's not the opposite of Merge.
func (s *Set) Separate(t *Set) {
	s.Remove(t.List()...)
}

// Union is the merger of multiple sets. It returns a new set with all the
// elements present in all the sets that are passed.
func Union(set1, set2 *Set, sets ...*Set) *Set {
	u := set1.Copy()
	set2.Each(func(item uint32) bool {
		u.Add(item)
		return true
	})
	for _, set := range sets {
		set.Each(func(item uint32) bool {
			u.Add(item)
			return true
		})
	}
	return u
}

// Difference returns a new set which contains items which are in in the first
// set but not in the others.
func Difference(set1, set2 *Set, sets ...*Set) *Set {
	s := set1.Copy()
	s.Separate(set2)
	for _, set := range sets {
		s.Separate(set) // separate is thread safe
	}
	return s
}

// Intersection returns a new set which contains items that only exist in all given sets.
func Intersection(set1, set2 *Set, sets ...*Set) *Set {
	all := Union(set1, set2, sets...)
	result := Union(set1, set2, sets...)

	all.Each(func(item uint32) bool {
		if !set1.Has(item) || !set2.Has(item) {
			result.Remove(item)
		}

		for _, set := range sets {
			if !set.Has(item) {
				result.Remove(item)
			}
		}
		return true
	})
	return result
}

// SymmetricDifference returns a new set which s is the difference of items which are in
// one of either, but not in both.
func SymmetricDifference(s *Set, t *Set) *Set {
	u := Difference(s, t)
	v := Difference(t, s)
	return Union(u, v)
}
