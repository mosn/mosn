# gohessian

[![Build Status](https://travis-ci.org/apache/dubbo-go-hessian2.png?branch=master)](https://travis-ci.org/apache/dubbo-go-hessian2)
[![codecov](https://codecov.io/gh/apache/dubbo-go-hessian2/branch/master/graph/badge.svg)](https://codecov.io/gh/apache/dubbo-go-hessian2)
[![GoDoc](https://godoc.org/github.com/apache/dubbo-go-hessian2?status.svg)](https://godoc.org/github.com/apache/dubbo-go-hessian2)
[![Go Report Card](https://goreportcard.com/badge/github.com/apache/dubbo-go-hessian2)](https://goreportcard.com/report/github.com/apache/dubbo-go-hessian2)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

---

It's a golang hessian library used by [Apache/dubbo-go](https://github.com/apache/dubbo-go).

## Feature List

* [All JDK Exceptions](https://github.com/apache/dubbo-go-hessian2/issues/59)
* [Field Alias By Alias](https://github.com/apache/dubbo-go-hessian2/issues/19)
* [Java Bigdecimal](https://github.com/apache/dubbo-go-hessian2/issues/89)
* [Java Date & Time](https://github.com/apache/dubbo-go-hessian2/issues/90)
* [Java Generic Invokation](https://github.com/apache/dubbo-go-hessian2/issues/84)
* [Java Extends](https://github.com/apache/dubbo-go-hessian2/issues/157)
* [Dubbo Attachements](https://github.com/apache/dubbo-go-hessian2/issues/49)
* [Skipping unregistered POJO](https://github.com/apache/dubbo-go-hessian2/pull/128)
* [Emoji](https://github.com/apache/dubbo-go-hessian2/issues/129)

## hessian type mapping between Java and Go

Cross languages message definition should be careful, the following situations should be avoided:

- define object that only exists in a special language
- using various java exceptions (using error code/message instead)

So we can maintain a cross language type mapping:

| hessian type |  java type  |  golang type | 
| --- | --- | --- | 
| **null** | null | nil | 
| **binary** | byte[] | []byte | 
| **boolean** | boolean | bool |
| **date** | java.util.Date | time.Time |
| **double** | double | float64 |
| **int** | int | int32 |
| **long** | long | int64 |
| **string** | java.lang.String | string |
| **list** | java.util.List | slice |
| **map** | java.util.Map | map |
| **object** | custom define object | custom define struct|
| **OTHER COMMON USING TYPE** | | | 
| **big decimal** | java.math.BigDecimal | github.com/dubbogo/gost/math/big/Decimal |
| **big integer** | java.math.BigInteger | github.com/dubbogo/gost/math/big/Integer |

## reference

- [hessian serialization](http://hessian.caucho.com/doc/hessian-serialization.html)

## Basic Usage Examples

### Encode To Bytes

```go
type Circular struct {
	Value
	Previous *Circular
	Next     *Circular
}

type Value struct {
	Num int
}

func (Circular) JavaClassName() string {
	return "com.company.Circular"
}

c := &Circular{}
c.Num = 12345
c.Previous = c
c.Next = c

e := NewEncoder()
err := e.Encode(c)
if err != nil {
    panic(err)
}

bytes := e.Buffer()
```

### Decode From Bytes

```go
decodedObject, err := NewDecoder(bytes).Decode()
if err != nil {
    panic(err)
}
circular, ok := obj.(*Circular)
// ...
```

## Customize Usage Examples

#### Encoding filed name

Hessian encoder default converts filed names of struct to lower camelcase, but you can customize it using `hessian` tag.

Example:
```go
type MyUser struct {
	UserFullName      string   `hessian:"user_full_name"`
	FamilyPhoneNumber string   // default convert to => familyPhoneNumber
}

func (MyUser) JavaClassName() string {
	return "com.company.myuser"
}

user := &MyUser{
    UserFullName:      "username",
    FamilyPhoneNumber: "010-12345678",
}

e := hessian.NewEncoder()
err := e.Encode(user)
if err != nil {
    panic(err)
}
```

The encoded bytes of the struct `MyUser` is as following:
```text
 00000000  43 12 63 6f 6d 2e 63 6f  6d 70 61 6e 79 2e 6d 79  |C.com.company.my|
 00000010  75 73 65 72 92 0e 75 73  65 72 5f 66 75 6c 6c 5f  |user..user_full_|
 00000020  6e 61 6d 65 11 66 61 6d  69 6c 79 50 68 6f 6e 65  |name.familyPhone|
 00000030  4e 75 6d 62 65 72 60 08  75 73 65 72 6e 61 6d 65  |Number`.username|
 00000040  0c 30 31 30 2d 31 32 33  34 35 36 37 38           |.010-12345678|
```

#### Decoding filed name

Hessian decoder finds the correct target field though comparing all filed names of struct one by one until matching.

The following example shows the order of the matching rules:
```go
type MyUser struct {
	MobilePhone      string   `hessian:"mobile-phone"`
}

// You must define the tag of struct for lookup filed form encoded binary bytes, in this case：
// 00000000  43 12 63 6f 6d 2e 63 6f  6d 70 61 6e 79 2e 6d 79  |C.com.company.my|
// 00000010  75 73 65 72 91 0c 6d 6f  62 69 6c 65 2d 70 68 6f  |user..mobile-pho|
// 00000020  6e 65 60 0b 31 37 36 31  32 33 34 31 32 33 34     |ne`.17612341234|
//
// mobile-phone(tag lookup) => mobilePhone(lowerCameCase) => MobilePhone(SameCase) => mobilephone(lowercase)
// ^ will matched


type MyUser struct {
	MobilePhone      string
}

// The following encoded binary bytes will be hit automatically:
//
// 00000000  43 12 63 6f 6d 2e 63 6f  6d 70 61 6e 79 2e 6d 79  |C.com.company.my|
// 00000010  75 73 65 72 91 0b 6d 6f  62 69 6c 65 50 68 6f 6e  |user..mobilePhon|
// 00000020  65 60 0b 31 37 36 31 32  33 34 31 32 33 34        |e`.17612341234|
//
// mobile-phone(tag lookup) => mobilePhone(lowerCameCase) => MobilePhone(SameCase) => mobilephone(lowercase)
//                             ^ will matched
//
// 00000000  43 12 63 6f 6d 2e 63 6f  6d 70 61 6e 79 2e 6d 79  |C.com.company.my|
// 00000010  75 73 65 72 91 0b 4d 6f  62 69 6c 65 50 68 6f 6e  |user..MobilePhon|
// 00000020  65 60 0b 31 37 36 31 32  33 34 31 32 33 34        |e`.17612341234|
//
// mobile-phone(tag lookup) => mobilePhone(lowerCameCase) => MobilePhone(SameCase) => mobilephone(lowercase)
//                                                           ^ will matched
//
// 00000000  43 12 63 6f 6d 2e 63 6f  6d 70 61 6e 79 2e 6d 79  |C.com.company.my|
// 00000010  75 73 65 72 91 0b 6d 6f  62 69 6c 65 70 68 6f 6e  |user..mobilephon|
// 00000020  65 60 0b 31 37 36 31 32  33 34 31 32 33 34        |e`.17612341234|
//
// mobile-phone(tag lookup) => mobilePhone(lowerCameCase) => MobilePhone(SameCase) => mobilephone(lowercase)
//                                                                                    ^ will matched

```


##### hessian.SetTagIdentifier

You can use `hessian.SetTagIdentifier` to customize tag-identifier of hessian, which takes effect to both encoder and decoder.

Example:

```go
hessian.SetTagIdentifier("json")

type MyUser struct {
	UserFullName      string   `json:"user_full_name"`
	FamilyPhoneNumber string   // default convert to => familyPhoneNumber
}

func (MyUser) JavaClassName() string {
	return "com.company.myuser"
}

user := &MyUser{
    UserFullName:      "username",
    FamilyPhoneNumber: "010-12345678",
}

e := hessian.NewEncoder()
err := e.Encode(user)
if err != nil {
    panic(err)
}
```

The encoded bytes of the struct `MyUser` is as following:

```text
 00000000  43 12 63 6f 6d 2e 63 6f  6d 70 61 6e 79 2e 6d 79  |C.com.company.my|
 00000010  75 73 65 72 92 0e 75 73  65 72 5f 66 75 6c 6c 5f  |user..user_full_|
 00000020  6e 61 6d 65 11 66 61 6d  69 6c 79 50 68 6f 6e 65  |name.familyPhone|
 00000030  4e 75 6d 62 65 72 60 08  75 73 65 72 6e 61 6d 65  |Number`.username|
 00000040  0c 30 31 30 2d 31 32 33  34 35 36 37 38           |.010-12345678|
```

#### Using Java collections
By default, the output of Hessian Java impl of a Java collection like java.util.HashSet will be decoded as `[]interface{}` in `go-hessian2`.
To apply the one-to-one mapping relationship between certain Java collection class and your Go struct, examples are as follows:

```go
//use HashSet as example
//define your struct, which should implements hessian.JavaCollectionObject
type JavaHashSet struct {
	value []interface{}
}

//get the inside slice value
func (j *JavaHashSet) Get() []interface{} {
	return j.value
}

//set the inside slice value
func (j *JavaHashSet) Set(v []interface{}) {
	j.value = v
}

//should be the same as the class name of the Java collection 
func (j *JavaHashSet) JavaClassName() string {
	return "java.util.HashSet"
}

func init() {
        //register your struct so that hessian can recognized it when encoding and decoding 
	SetCollectionSerialize(&JavaHashSet{})
}
```



## Notice for inheritance

`go-hessian2` supports inheritance struct, but the following situations should be avoided.

+ **Avoid fields with the same name in multiple parent struct**

The following struct `C` have inherited field `Name`(default from the first parent), 
but it's confused in logic.

```go
type A struct { Name string }
type B struct { Name string }
type C struct {
	A
	B
}
```

+ **Avoid inheritance for a pointer of struct**

The following definition is valid for golang syntax, 
but the parent will be nil when create a new Dog, like `dog := Dog{}`, 
which will not happen in java inheritance, 
and is also not supported by `go-hessian2`.

```go
type Dog struct {
	*Animal
}
```