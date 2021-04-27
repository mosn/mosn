# SuperMonkey

![gopher](https://user-images.githubusercontent.com/29589055/102566226-0a23b700-411a-11eb-879e-8b58b6a1c8d6.png)

This lib is inspired by https://github.com/bouk/monkey, and uses some of the code

## Introduction

Patch all functions without limits, including which are unexported

**Warning** : please add -l to your gcflags or add `//go:noinline` to func which you want to patch.


## when running in tests

you should run this lib under a go mod project and provide the full project path

**Warning** : use `go test -ldflags="-s=false" -gcflags="-l"` to enable symbol table and disable inline.

## when running not in tests

### patch private function

#### normal

```go
package main

import (
	"fmt"

	sm "github.com/cch123/supermonkey"
)

func main() {
	fmt.Println("original function output:")
	heyHey()

	patchGuard := sm.Patch(heyHey, func() {
		fmt.Println("please be polite")
	})
	fmt.Println("after patch, function output:")
	heyHey()

	patchGuard.Unpatch()
	fmt.Println("unpatch, then output:")
	heyHey()
}

//go:noinline
func heyHey() {
	fmt.Println("fake")
}
```

> go run -gcflags="-l" yourfile.go

#### full symbol name

```go
package main

import (
	"fmt"

	sm "github.com/cch123/supermonkey"
)

func main() {
	fmt.Println("original function output:")
	heyHeyHey()

	patchGuard := sm.PatchByFullSymbolName("main.heyHeyHey", func() {
		fmt.Println("please be polite")
	})
	fmt.Println("after patch, function output:")
	heyHeyHey()

	patchGuard.Unpatch()
	fmt.Println("unpatch, then output:")
	heyHeyHey()
}

//go:noinline
func heyHeyHey() {
	fmt.Println("fake")
}
```

> go run -gcflags="-l" yourfile.go

### patch private instance method

#### normal

```go
package main

import (
	"fmt"

	sm "github.com/cch123/supermonkey"
)

type person struct{ name string }

//go:noinline
func (p *person) speak() {
	fmt.Println("my name is ", p.name)
}

func main() {
	var p = person{"Lance"}
	fmt.Println("original function output:")
	p.speak()

	patchGuard := sm.Patch((*person).speak, func(*person) {
        fmt.Println("we are all the same")
    })
    fmt.Println("after patch, function output:")
    p.speak()

	patchGuard.Unpatch()
	fmt.Println("unpatch, then output:")
	p.speak()
}
```

> go run -gcflags="-l" yourfile.go

#### full symbol name

```go
package main

import (
	"fmt"
	"unsafe"

	sm "github.com/cch123/supermonkey"
)

type person struct{ name string }

//go:noinline
func (p *person) speak() {
	fmt.Println("my name is ", p.name)
}

func main() {
	var p = person{"Linda"}
	fmt.Println("original function output:")
	p.speak()

	patchGuard := sm.PatchByFullSymbolName("main.(*person).speak", func(ptr uintptr) {
		p = (*person)(unsafe.Pointer(ptr))
		fmt.Println(p.name, ", we are all the same")
	})
	fmt.Println("after patch, function output:")
	p.speak()

	patchGuard.Unpatch()
	fmt.Println("unpatch, then output:")
	p.speak()
}
```

```go
package main

import (
	"context"
	"fmt"

	sm "github.com/cch123/supermonkey"
)

type Bar struct {
	Name string
}

type Foo struct{}

func (*Foo) MyFunc(ctx context.Context) (*Bar, error) {
	return &Bar{"Bar"}, nil
}

func main() {
	f := &Foo{}
	fmt.Println("original function output:")
	fmt.Println(f.MyFunc(nil))

	//patchGuard := sm.PatchByFullSymbolName("main.(*Foo).MyFunc", func(_ *Foo, ctx context.Context) (*Bar, error) {
	patchGuard := sm.PatchByFullSymbolName("main.(*Foo).MyFunc", func(_ uintptr, ctx context.Context) (*Bar, error) {
		return &Bar{"Not bar"}, nil
	})

	fmt.Println("after patch, function output:")
	fmt.Println(f.MyFunc(nil))

	patchGuard.Unpatch()
	fmt.Println("unpatch, then output:")
	fmt.Println(f.MyFunc(nil))
}
```

> go run -gcflags="-l" yourfile.go

## Authors

[cch123](https://github.com/cch123)

[Mutated1994](https://github.com/Mutated1994)

