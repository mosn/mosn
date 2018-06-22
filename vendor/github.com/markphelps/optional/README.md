# Optional

[![Build Status](https://travis-ci.org/markphelps/optional.svg?branch=master)](https://travis-ci.org/markphelps/optional)
[![Release](https://img.shields.io/github/release/markphelps/optional.svg?style=flat-square)](https://github.com/markphelps/optional/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square)](LICENSE.md)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/markphelps/optional)
[![Go Report Card](https://goreportcard.com/badge/github.com/markphelps/optional?style=flat-square)](https://goreportcard.com/report/github.com/markphelps/optional)
[![SayThanks.io](https://img.shields.io/badge/SayThanks.io-%E2%98%BC-1EAEDB.svg?style=flat-square)](https://saythanks.io/to/markphelps)

Optional is a tool that generates 'optional' type wrappers around a given type T.

It is also a library that provides optional types for the primitive Go types.

## Motivation

Ever had to write a test where you want to assert something only if a value is present?

```go
tests :=  []struct {
  data string
  dataPresent bool
} {
    { "YOLO kombucha slow-carb wayfarers fixie", true },
    { "", false },
}

...

if test.dataPresent {
  assert.Equal(t, hipsterism, test.data)
}
```

Now you can simplify all that with `optional` types:

```go
tests :=  []struct {
  data optional.String
} {
    { optional.NewString("viral narwhal etsy twee VHS") },
    { optional.String{} },
  }
}

...

test.data.If(func(s string) {
    assert.Equal(t, hipsterism, s)
})
```

## Marshalling/Unmarshalling JSON

Optional types also marshal to/from JSON as you would expect:

### Marshalling

```go
func main() {
  var value = struct {
    Field optional.String `json:"field,omitempty"`
  }{
    Field: optional.NewString("bar"),
  }

  out, _ := json.Marshal(value)
  fmt.Println(string(out))
  // outputs: {"field":"bar"}
}
```

### Unmarshalling

```go
func main() {
  var value = &struct {
    Field optional.String `json:"field,omitempty"`
  }{}

  _ = json.Unmarshal([]byte(`{"field":"bar"}`), value)

  value.Field.If(func(s string) {
    fmt.Println(s)
  })
  // outputs: bar
}
```

See [example_test.go](example_test.go) for more examples.

## Inspiration

* Java [Optional](https://docs.oracle.com/javase/8/docs/api/java/util/Optional.html)
* [https://github.com/leighmcculloch/go-optional](https://github.com/leighmcculloch/go-optional)
* [https://github.com/golang/go/issues/7054](https://github.com/golang/go/issues/7054)

## Tool

### Install

`go get -u github.com/markphelps/optional/cmd/optional`

### Usage

Typically this process would be run using go generate, like this:

```go
//go:generate optional -type=Foo
```

running this command:

```bash
optional -type=Foo
```

in the same directory will create the file optional_foo.go
containing a definition of:

```go
type OptionalFoo struct {
  ...
}
```

The default type is OptionalT or optionalT (depending on if the type is exported)
and output file is optional_t.go. This can be overridden with the -output flag.

## Library

* [bool](bool.go)
* [byte](byte.go)
* [complex128](complex128.go)
* [complex64](complex64.go)
* [float32](float32.go)
* [float64](float64.go)
* [int](int.go)
* [int16](int16.go)
* [int32](int32.go)
* [int64](int64.go)
* [int8](int8.go)
* [rune](rune.go)
* [string](string.go)
* [uint](uint.go)
* [uint16](uint16.go)
* [uint32](uint32.go)
* [uint64](uint64.go)
* [uint8](uint8.go)
* [uintptr](uintptr.go)
* [error](error.go)

### Usage

```go
import (
  "fmt"

  "github.com/markphelps/optional"
)

s := optional.NewString("foo")

if s.Present() {
  value, _ := s.Get()
  fmt.Println(value)
}

t := optional.String{}
fmt.Println(t.OrElse("bar"))
```

See [example_test.go](example_test.go) and the [documentation](http://godoc.org/github.com/markphelps/optional) for more usage.

## Contributing

1. [Fork it](https://github.com/markphelps/optional/fork)
1. Create your feature branch (`git checkout -b my-new-feature`)
1. Commit your changes (`git commit -am 'Add some feature'`)
1. Push to the branch (`git push origin my-new-feature`)
1. Create a new Pull Request
