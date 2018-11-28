package gxgzip

import (
	"testing"
)

func TestDoGzipCompress(t *testing.T) {
	var str = "hello, world"
	zipStr := DoGzipCompress([]byte(str))
	uncompressString, err := DoGzipUncompress(zipStr)
	if string(uncompressString) != str || err != nil {
		t.Errorf("str:%q, uncompress string:%q, err:%s", str, string(uncompressString), err)
	}
}
