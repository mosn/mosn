package gxzlib

import "testing"

func TestDoZlibCompress(t *testing.T) {
	var str = "hello, world"
	zipStr := DoZlibCompress([]byte(str))
	uncompressString, err := DoZlibUncompress(zipStr)
	if string(uncompressString) != str || err != nil {
		t.Errorf("str:%q, uncompress string:%q, err:%s", str, string(uncompressString), err)
	}
}
