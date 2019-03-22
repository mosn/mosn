package set

import (
	"testing"
)

func TestSet(t *testing.T) {
	s := NewSet("key")
	list := s.Slice()
	if int32(len(list)) == s.Len() {
		t.Log("success:", list)
	} else {
		t.Error("fail")
	}

	e2 := "key"
	s.Add(e2)
	if s.Len() == 1 {
		t.Log("Add success:", s.Slice())
	} else {
		t.Error("add fail", s.Slice())
	}

	e3 := "key2"
	s.Add(e3)
	if s.Len() == 2 {
		t.Log("Add success:", s.Slice())
	} else {
		t.Error("add fail", s.Slice())
	}

	s.Remove(e3)
	if s.Len() == 1 {
		t.Log("remove success:", s.Slice())
	} else {
		t.Error("remove fail", s.Slice())
	}

	if s.Has("key") {
		t.Log("has key success")
	} else {
		t.Error("has error")
	}

	if s.Has("key2333") {
		t.Error("has error")
	} else {
		t.Log("has key success")
	}

	s.Clear()
	if s.Len() == 0 {
		t.Log("clear success:", s.Slice())
	} else {
		t.Error("clear fail", s.Slice())
	}

}

func BenchmarkSet_add(b *testing.B) {
	s := new(Set)
	for i := 0; i < b.N; i++ {
		s.Add(i)
	}
}

func BenchmarkSet_Slice(b *testing.B) {
	s := NewSet(1, 2, 3)
	for i := 0; i < b.N; i++ {
		s.Slice()
	}
}
