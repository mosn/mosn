package setmultimap_test

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jwangsadinata/go-multimap/setmultimap"
)

// The following examples is to demonstrate basic usage of MultiMap.
func ExampleMultiMap_Clear() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Clear the current map.
	m.Clear() // empty

	// Verify that it is empty.
	fmt.Printf("%v\n", m.Empty())
	fmt.Printf("%v\n", m.Size())

	// Output:
	// true
	// 0
}

func ExampleMultiMap_Contains() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Check whether the multimap contains a certain key/value pair.
	found := m.Contains(1, "a") // true
	fmt.Printf("%v\n", found)

	found = m.Contains(1, "b") // false
	fmt.Printf("%v\n", found)

	found = m.Contains(2, "b") // true
	fmt.Printf("%v\n", found)

	// Output:
	// true
	// false
	// true
}

func ExampleMultiMap_ContainsKey() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Check whether the multimap contains a certain key.
	found := m.ContainsKey(1) // true
	fmt.Printf("%v\n", found)

	found = m.ContainsKey(2) // true
	fmt.Printf("%v\n", found)

	found = m.ContainsKey(3) // true
	fmt.Printf("%v\n", found)

	// Output:
	// true
	// true
	// false
}

func ExampleMultiMap_ContainsValue() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Check whether the multimap contains a certain value.
	found := m.ContainsValue("a") // true
	fmt.Printf("%v\n", found)

	found = m.ContainsValue("b") // true
	fmt.Printf("%v\n", found)

	found = m.ContainsValue("c") // true
	fmt.Printf("%v\n", found)

	// Output:
	// true
	// true
	// false
}

func ExampleMultiMap_Get() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Retrieve values from the map.
	value, found := m.Get(3) // nil, false
	fmt.Printf("%v, %v\n", value, found)

	value, found = m.Get(2) // {b}, true
	fmt.Printf("%v, %v\n", value, found)

	value, found = m.Get(1) // {a, x}, true (random order)

	// Workaround for test output consistency:
	tmp := make([]string, len(value))
	count := 0
	for _, s := range value {
		tmp[count] = s.(string)
		count++
	}
	sort.Strings(tmp)
	fmt.Printf("%v, %v\n", tmp, found)

	// Output:
	// [], false
	// [b], true
	// [a x], true
}

func ExampleMultiMap_Put() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Verify that the map has the right size.
	fmt.Println(m.Size())

	// Output:
	// 3
}

func ExampleMultiMap_Entries() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Retrieve all the keys in the map.
	entries := m.Entries() // {1,a}, {1,x}, {2,b} (random order)

	// Workaround for test output consistency.
	tmp := make([]struct {
		Key   int
		Value string
	}, len(entries))
	count := 0
	for _, e := range entries {
		tmp[count] = struct {
			Key   int
			Value string
		}{e.Key.(int), e.Value.(string)}
		count++
	}
	sort.Sort(byKeyThenValue(tmp))
	fmt.Printf("%v\n", tmp)

	// Output:
	// [{1 a} {1 x} {2 b}]
}

func ExampleMultiMap_Keys() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Retrieve all the keys in the multimap.
	keys := m.Keys() // 1, 1, 2 (random order)

	// Workaround for test output consistency.
	tmp := make([]int, len(keys))
	count := 0
	for _, key := range keys {
		tmp[count] = key.(int)
		count++
	}
	sort.Ints(tmp)
	fmt.Printf("%v\n", tmp)

	// Output:
	// [1 1 2]
}

func ExampleMultiMap_KeySet() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Retrieve all the distinct keys in the multimap.
	keys := m.KeySet() // 1, 2  (random order)

	// Workaround for test output consistency.
	tmp := make([]int, len(keys))
	count := 0
	for _, key := range keys {
		tmp[count] = key.(int)
		count++
	}
	sort.Ints(tmp)
	fmt.Printf("%v\n", tmp)

	// Output:
	// [1 2]
}

func ExampleMultiMap_Values() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Retrieve all the keys in the map.
	values := m.Values() // a, b, x  (random order)

	// Workaround for test output consistency.
	tmp := make([]string, len(values))
	count := 0
	for _, value := range values {
		tmp[count] = value.(string)
		count++
	}
	sort.Strings(tmp)
	fmt.Printf("%v\n", tmp)

	// Output:
	// [a b x]
}

func ExampleMultiMap_Remove() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Remove a key/value pair from the multimap.
	m.Remove(2, "b")

	// Verify that the map has less number of key/value pair.
	fmt.Println(m.Size()) // 2

	// Also, verify that there are no more (2, "b") key/value pair.
	value, found := m.Get(2)
	fmt.Printf("%v, %v", value, found) // nil, false

	// Output:
	// 2
	// [], false
}

func ExampleMultiMap_RemoveAll() {
	// Create a new multimap
	m := setmultimap.New() // empty

	// Put some contents to the multimap.
	m.Put(1, "x") // 1->x
	m.Put(2, "b") // 1->x, 2->b
	m.Put(1, "a") // 1->a, 1->x, 2->b

	// Remove a key/value pair from the multimap.
	m.RemoveAll(1)

	// Verify that the map has less number of key/value pair.
	fmt.Println(m.Size()) // 1

	// Also, verify that there are no more values with key 1.
	value, found := m.Get(1)
	fmt.Printf("%v, %v", value, found) // nil, false

	// Output:
	// 1
	// [], false
}

// Helper for sorting entries.
type byKeyThenValue []struct {
	Key   int
	Value string
}

func (a byKeyThenValue) Len() int      { return len(a) }
func (a byKeyThenValue) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byKeyThenValue) Less(i, j int) bool {
	if a[i].Key < a[j].Key {
		return true
	} else if a[i].Key > a[j].Key {
		return false
	} else {
		return strings.Compare(a[i].Value, a[j].Value) < 0
	}
}
