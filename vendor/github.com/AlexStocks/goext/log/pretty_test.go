/* log_test.go - test for log.go */
package gxlog

import (
	"fmt"
	"testing"
)

type info struct {
	name string
	age  float32
	m    map[string]string
}

func TestPrettyString(t *testing.T) {
	var i = info{name: "hello", age: 23.5, m: map[string]string{"h": "w", "hello": "world"}}
	fmt.Println(PrettyString(i))
}

func TestColorPrint(t *testing.T) {
	var i = info{name: "hello", age: 23.5, m: map[string]string{"h": "w", "hello": "world"}}
	ColorPrintln(i)
}

func TestColorPrintf(t *testing.T) {
	var i = info{name: "hello", age: 23.5, m: map[string]string{"h": "w", "hello": "world"}}
	ColorPrintf("exapmle format:%s\n", i)
}
