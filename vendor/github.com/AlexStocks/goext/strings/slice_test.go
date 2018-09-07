package gxstrings

import (
	"testing"
)

// go test  -v slice_test.go  slice.go

// slice转string之后，如果slice的值有变化，string也会跟着改变
func TestString(t *testing.T) {
	b := []byte("hello world")
	a := String(b)
	b[0] = 'a'
	println(a) //output  aello world
}

func TestSlice(t *testing.T) {
	// 编译器会把 "hello, world" 这个字符串常量的字节数组分配在没有写权限的 memory segment
	a := "hello world"
	b := Slice(a)

	// !!! 上面这个崩溃在defer里面是recover不回来的，真的就崩溃了，原因可能就跟c的非法内存访问一样，os不跟你玩了
	// b[0] = 'a' //这里就等着崩溃吧

	//但是可以这样，因为go又重新给b分配了内存
	b = append(b, "hello world"...)
	println(String(b)) // output: hello worldhello world
}

func TestSlice2(t *testing.T) {
	// 你可以动态生成一个字符串，使其分配在可写的区域，比如 gc heap，那么就不会崩溃了。
	a := string([]byte("hello world"))
	b := Slice(a)
	b = append(b, "hello world"...)
	println(String(b)) // output: hello worldhello world
}

// func TestCheckByteArray(t *testing.T) {
// 	var s = "hello"
// 	var flag bool
// 	if flag = CheckByteArray1([]byte(s)); !flag {
// 		t.Fatalf("CheckByteArray([]byte(%s)) failed")
// 	}
//
// 	if flag = CheckByteArray1(s); flag {
// 		t.Fatalf("CheckByteArray(%s) failed")
// 	}
// }

// // go test -v -bench BenchmarkCheckByteArray1$ -run=^a
// func BenchmarkCheckByteArray1(b *testing.B) {
// 	var s = []byte("hello")
// 	for i := 0; i < b.N; i++ {
// 		CheckByteArray1(s)
// 	}
// }
//
// // go test -v -bench BenchmarkCheckByteArray2$ -run=^a
// func BenchmarkCheckByteArray2(b *testing.B) {
// 	var s = []byte("hello")
// 	for i := 0; i < b.N; i++ {
// 		CheckByteArray2(s)
// 	}
// }
//
// go test -bench=. -run=CheckByteArray$
// BenchmarkCheckByteArray-4    	20000000	        60.6 ns/op
// BenchmarkCheckByteArray2-4   	2000000000	         1.02 ns/op
// PASS
// ok  	github.com/AlexStocks/goext/strings	3.427s
