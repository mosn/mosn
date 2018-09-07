package main

import (
	"fmt"
	"strings"
)

import (
	"github.com/AlexStocks/goext/io/kv/disk"
)

func AdvancedTransformExample(key string) *gxdiskv.PathKey {
	path := strings.Split(key, "/")
	last := len(path) - 1
	return &gxdiskv.PathKey{
		Path:     path[:last],
		FileName: path[last] + ".txt",
	}
}

// If you provide an AdvancedTransform, you must also provide its
// inverse:

func InverseTransformExample(pathKey *gxdiskv.PathKey) (key string) {
	txt := pathKey.FileName[len(pathKey.FileName)-4:]
	if txt != ".txt" {
		panic("Invalid file found in storage folder!")
	}
	return strings.Join(pathKey.Path, "/") + pathKey.FileName[:len(pathKey.FileName)-4]
}

func main() {
	d := gxdiskv.New(gxdiskv.Options{
		BasePath:          "my-data-dir",
		AdvancedTransform: AdvancedTransformExample,
		InverseTransform:  InverseTransformExample,
		CacheSizeMax:      1024 * 1024,
	})
	// Write some text to the key "alpha/beta/gamma".
	key := "alpha/beta/gamma"
	d.WriteString(key, "¡Hola!") // will be stored in "<basedir>/alpha/beta/gamma.txt"
	fmt.Println(d.ReadString("alpha/beta/gamma"))
}
