package gxstrings

import (
	"testing"
)

// go test -v -timeout 30s github.com/AlexStocks/goext/strings -run TestMerge
func TestMerge(t *testing.T) {
	s1 := []string{"hello0", "hello1"}
	s2 := []string{}

	s3 := Merge(s1, s2)
	if len(s3) != 2 {
		t.Errorf("the length of s3 should be 2")
	}
	t.Logf("s3 %#v", s3)

	s1 = []string{"hello0", "hello1"}
	s2 = []string{"world"}

	s3 = Merge(s1, s2)
	if len(s3) != 3 {
		t.Errorf("the length of s3 should be 3")
	}
	t.Logf("s3 %#v", s3)

}
