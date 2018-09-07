# Set [![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/scylladb/go-set) [![Go Report Card](https://goreportcard.com/badge/github.com/scylladb/go-set)](https://goreportcard.com/report/github.com/scylladb/go-set) [![Build Status](https://travis-ci.org/scylladb/go-set.svg?branch=master)](https://travis-ci.org/scylladb/go-set)

Package set is a type-safe, zero-allocation port of the excellent package [fatih/set](https://github.com/fatih/set). It contains sets for most of the basic types and you can generate set for your own types with ease.

## Example

Example code using the generated string set:

```go
import "github.com/scylladb/go-set/strset"

s1 := strset.New("entry 1", "entry 2")
s2 := strset.New("entry 2", "entry 3")
s3 := strset.Intersection(s1, s2)
// s3 now contains only "entry 2"
```

The library exposes a number of top level factory functions that can be used to create a specific instances of the set type you want to use. For example to create a set to store `int` you could do like this:

```go
import "github.com/scylladb/go-set"

s := set.NewIntSet()
// use the set...
```

## Usage

In every subpackage Set is the main set structure that holds all the data and methods used to working with the set.

#### func  Difference

```go
func Difference(set1, set2 *Set, sets ...*Set) *Set
```
Difference returns a new set which contains items which are in in the first set
but not in the others.

#### func  Intersection

```go
func Intersection(set1, set2 *Set, sets ...*Set) *Set
```
Intersection returns a new set which contains items that only exist in all given
sets.

#### func  New

```go
func New(ts ...int) *Set
```
New creates and initializes a new Set interface. Its single parameter denotes the
type of set to create. Either ThreadSafe or NonThreadSafe. The default is
ThreadSafe.

#### func  SymmetricDifference

```go
func SymmetricDifference(s *Set, t *Set) *Set
```
SymmetricDifference returns a new set which s is the difference of items which
are in one of either, but not in both.

#### func  Union

```go
func Union(set1, set2 *Set, sets ...*Set) *Set
```
Union is the merger of multiple sets. It returns a new set with all the elements
present in all the sets that are passed.

#### func (*Set) Add

```go
func (s *Set) Add(items ...int)
```
Add includes the specified items (one or more) to the Set. The underlying Set s
is modified. If passed nothing it silently returns.

#### func (*Set) Clear

```go
func (s *Set) Clear()
```
Clear removes all items from the Set.

#### func (*Set) Copy

```go
func (s *Set) Copy() *Set
```
Copy returns a new Set with a copy of s.

#### func (*Set) Each

```go
func (s *Set) Each(f func(item int) bool)
```
Each traverses the items in the Set, calling the provided function for each Set
member. Traversal will continue until all items in the Set have been visited, or
if the closure returns false.

#### func (*Set) Has

```go
func (s *Set) Has(items ...int) bool
```
Has looks for the existence of items passed. It returns false if nothing is
passed. For multiple items it returns true only if all of the items exist.

#### func (*Set) IsEmpty

```go
func (s *Set) IsEmpty() bool
```
IsEmpty reports whether the Set is empty.

#### func (*Set) IsEqual

```go
func (s *Set) IsEqual(t *Set) bool
```
IsEqual test whether s and t are the same in size and have the same items.

#### func (*Set) IsSubset

```go
func (s *Set) IsSubset(t *Set) (subset bool)
```
IsSubset tests whether t is a subset of s.

#### func (*Set) IsSuperset

```go
func (s *Set) IsSuperset(t *Set) bool
```
IsSuperset tests whether t is a superset of s.

#### func (*Set) List

```go
func (s *Set) List() []int
```
List returns a slice of all items. There is also StringSlice() and IntSlice()
methods for returning slices of type string or int.

#### func (*Set) Merge

```go
func (s *Set) Merge(t *Set)
```
Merge is like Union, however it modifies the current Set it's applied on with
the given t Set.

#### func (*Set) Pop

```go
func (s *Set) Pop() int
```
Pop deletes and return an item from the Set. The underlying Set s is modified.
If Set is empty, nil is returned.

#### func (*Set) Remove

```go
func (s *Set) Remove(items ...int)
```
Remove deletes the specified items from the Set. The underlying Set s is
modified. If passed nothing it silently returns.

#### func (*Set) Separate

```go
func (s *Set) Separate(t *Set)
```
Separate removes the Set items containing in t from Set s. Please aware that
it's not the opposite of Merge.

#### func (*Set) Size

```go
func (s *Set) Size() int
```
Size returns the number of items in a Set.

#### func (*Set) String

```go
func (s *Set) String() string
```
String returns a string representation of s
 
## Performance

The improvement in performance by using concrete types over `interface{}` is notable. Below you will find benchmark results comparing type-safe sets to `fatih/set` counterparts for `string`, `int64`, `int32`, `float64` and `float32` on a local machine, Intel(R) Core(TM) i7-7500U CPU @ 2.70GHz.

```
pkg: github.com/scylladb/go-set/strset
BenchmarkTypeSafeSetHasNonExisting-4            200000000                7.02 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasNonExisting-4           20000000                60.0 ns/op            32 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExisting-4               200000000                9.02 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExisting-4              20000000                97.0 ns/op            32 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExistingMany-4           100000000               16.8 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExistingMany-4          30000000               106 ns/op              32 B/op          2 allocs/op
BenchmarkTypeSafeSetAdd-4                        3000000               469 ns/op              58 B/op          0 allocs/op
BenchmarkInterfaceSetAdd-4                       2000000               909 ns/op             117 B/op          2 allocs/op

pkg: github.com/scylladb/go-set/i64set
BenchmarkTypeSafeSetHasNonExisting-4            300000000                5.51 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasNonExisting-4           30000000                49.4 ns/op            24 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExisting-4               200000000                7.53 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExisting-4              20000000                68.5 ns/op            24 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExistingMany-4           100000000               11.0 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExistingMany-4          20000000                74.5 ns/op            24 B/op          2 allocs/op
BenchmarkTypeSafeSetAdd-4                       10000000               225 ns/op              40 B/op          0 allocs/op
BenchmarkInterfaceSetAdd-4                       3000000               403 ns/op              82 B/op          2 allocs/op

pkg: github.com/scylladb/go-set/i32set
BenchmarkTypeSafeSetHasNonExisting-4            300000000                5.61 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasNonExisting-4           30000000                48.8 ns/op            20 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExisting-4               200000000                7.07 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExisting-4              20000000                69.3 ns/op            20 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExistingMany-4           100000000               11.7 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExistingMany-4          20000000                71.1 ns/op            20 B/op          2 allocs/op
BenchmarkTypeSafeSetAdd-4                       10000000               206 ns/op              25 B/op          0 allocs/op
BenchmarkInterfaceSetAdd-4                       3000000               394 ns/op              78 B/op          2 allocs/op

pkg: github.com/scylladb/go-set/f64set
BenchmarkTypeSafeSetHasNonExisting-4            300000000                5.82 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasNonExisting-4           30000000                49.8 ns/op            24 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExisting-4               50000000                26.8 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExisting-4              20000000                77.6 ns/op            24 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExistingMany-4           50000000                27.6 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExistingMany-4          20000000                82.3 ns/op            24 B/op          2 allocs/op
BenchmarkTypeSafeSetAdd-4                       10000000               270 ns/op              40 B/op          0 allocs/op
BenchmarkInterfaceSetAdd-4                       3000000               428 ns/op              82 B/op          2 allocs/op

pkg: github.com/scylladb/go-set/f32set
BenchmarkTypeSafeSetHasNonExisting-4            300000000                5.78 ns/op            0 B/op          0 allocs/op
BenchmarkInterfaceSetHasNonExisting-4           30000000                49.3 ns/op            20 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExisting-4               50000000                24.9 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExisting-4              20000000                78.6 ns/op            20 B/op          2 allocs/op
BenchmarkTypeSafeSetHasExistingMany-4           50000000                25.1 ns/op             0 B/op          0 allocs/op
BenchmarkInterfaceSetHasExistingMany-4          20000000                81.1 ns/op            20 B/op          2 allocs/op
BenchmarkTypeSafeSetAdd-4                       10000000               246 ns/op              24 B/op          0 allocs/op
BenchmarkInterfaceSetAdd-4                       3000000               408 ns/op              78 B/op          2 allocs/op
```

## Code generation
For code generation we use [Google go_generics tool](https://github.com/mmatczuk/go_generics) that we forked to provide bazel-free installation, to install run:

```bash
go get -u github.com/mmatczuk/go_generics/cmd/go_generics
go get -u github.com/mmatczuk/go_generics/cmd/go_merge
``` 

Once you have `go_generics` installed properly you can regenerate the code using `go generate` in the top level directory.

### Your custom types

If you have types that you would like to use but the are not amenable for inclusion in this library you can simply generate code on your own and put it in your package.

For example, to generate a set for `SomeType` in package `sometypeset` call:

```
go_generics -i internal/set/set.go -t T=SomeType -o path/to/outputfile.go -p sometypeset
go_generics -i internal/set/set_test.go -t P=SomeType -o path/to/outputfile_test.go -p sometypeset
```

If you think your addition belongs here we are open to accept pull requests.

## License

Copyright (C) 2018 ScyllaDB

This project is distributed under the Apache 2.0 license. See the [LICENSE](https://github.com/scylladb/gocqlx/blob/master/LICENSE) file for details.
It contains software from:

* [github.com/fatih/set](https://github.com/fatih/set), licensed under the MIT license

GitHub star is always appreciated!
