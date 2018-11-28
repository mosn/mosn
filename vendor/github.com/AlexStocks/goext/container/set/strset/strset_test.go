package strset

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/fatih/set"
)

func TestAdd(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()
	s.Add(e1)
	s.Add(e2)

	if len(s.m) != 2 {
		t.Errorf("expected 2 entries, got %d", len(s.m))
	}
}

func TestRemove(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()
	s.Add(e1)
	s.Add(e2)

	s.Remove(e1)

	if len(s.m) != 1 {
		t.Errorf("expected 1 entries, got %d", len(s.m))
	}

	if _, ok := s.m[e2]; !ok {
		t.Errorf("wrong entry %v removed, expected %v", e1, e2)
	}
}

func TestPop(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()
	popped := s.Pop()
	if popped != nonExistent {
		t.Errorf("default non existent sentinel not returned, instead got %v", popped)
	}

	s.Add(e1)
	s.Add(e2)

	s.Pop()

	if len(s.m) != 1 {
		t.Errorf("expected 1 entries, got %d", len(s.m))
	}
}

func TestHas(t *testing.T) {
	var e1, e2, e3 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e3 = v
	}

	s := New()
	if s.Has(e1) {
		t.Errorf("expected a new set to not contain %v", e1)
	}

	s.Add(e1)
	s.Add(e2)

	if !s.Has(e1) {
		t.Errorf("expected the set to contain %v", e1)
	}
	if !s.Has(e2) {
		t.Errorf("expected the set to contain %v", e2)
	}
	if s.Has(e3) {
		t.Errorf("did not expect the set to contain %v", e3)
	}
}

func TestSize(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()
	s.Add(e1)
	s.Add(e2)

	if s.Size() != 2 {
		t.Errorf("expected the set size to be 2 but it was %d", s.Size())
	}
}

func TestClear(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()
	s.Add(e1)
	s.Add(e2)

	s.Clear()

	if s.Size() != 0 {
		t.Errorf("expected the cleared set size to be 0 but it was %d", s.Size())
	}
}

func TestIsEmpty(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()
	if !s.IsEmpty() {
		t.Errorf("expected new set to be empty but it had %d elements", s.Size())
	}

	s.Add(e1)
	s.Add(e2)

	if s.IsEmpty() {
		t.Error("expected a set with added items to not be empty")
	}
}

func TestIsEqual(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s1 := New()
	s2 := New()
	if !s1.IsEqual(s2) {
		t.Error("expected 2 new sets to be equal")
	}

	s1.Add(e1)
	s1.Add(e2)
	if s1.IsEqual(s2) {
		t.Errorf("expected 2 different sets to be equal, %v, %v", s1, s2)
	}
	s2.Add(e1)
	s2.Add(e2)

	if !s1.IsEqual(s2) {
		t.Error("expected 2 sets with the same items added to be equal")
	}
}

func TestIsSubset(t *testing.T) {
	var e1, e2, e3 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e3 = v
	}

	s1 := New()
	s2 := New()

	s1.Add(e1)
	s1.Add(e2)
	s2.Add(e1)

	if !s1.IsSubset(s2) {
		t.Errorf("expected %v to be a subset of %v", s2, s1)
	}

	s2.Add(e2)
	s2.Add(e3)
	if s1.IsSubset(s2) {
		t.Errorf("expected %v not to be a subset of %v", s2, s1)
	}
}

func TestIsSuperset(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s1 := New()
	s2 := New()

	s1.Add(e1)
	s1.Add(e2)
	s2.Add(e1)

	if !s2.IsSuperset(s1) {
		t.Errorf("expected %v to be a super set of %v", s1, s2)
	}
}

func TestCopy(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s1 := New()

	s1.Add(e1)
	s1.Add(e2)

	s2 := s1.Copy()

	if !s2.IsEqual(s1) {
		t.Errorf("expected %v to be equal to %v after copy", s1, s2)
	}
}

func TestString(t *testing.T) {
	var e1 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}

	s1 := New()

	s1.Add(e1)

	s := s1.String()

	if s == "" {
		t.Errorf("expected string representation %v to exist", s)
	}

	if !strings.HasPrefix(s, "[") && !strings.HasSuffix(s, "]") {
		t.Errorf("expected string representation %v to start with '[' and end with ']'", s)
	}
}

func TestMerge(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s1 := New()
	s2 := New()

	s1.Add(e1)
	s2.Add(e2)

	s1.Merge(s2)

	if s1.Size() != 2 {
		t.Errorf("expected merged set %v have size 2 but it has %d", s1, s1.Size())
	}
}

func TestList(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s := New()

	s.Add(e1)
	s.Add(e2)

	l := s.List()

	if len(l) != s.Size() {
		t.Errorf("expected a set of size %d to give a list of size 2 but it has %d", s.Size(), len(l))
	}

	for _, e := range l {
		if !s.Has(e) {
			t.Errorf("listed entry %v not available in the set %s", e, s)
		}
	}
}

func TestSeparate(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s1 := New()
	s2 := New()

	s1.Add(e1)
	s1.Add(e2)

	s2.Add(e2)

	s1.Separate(s2)

	if s1.Size() != 1 {
		t.Errorf("expected separated set %v have size 1 but it has %d", s1, s1.Size())
	}

	if s1.Has(e2) {
		t.Errorf("separated set %v still contains %v", s1, e2)
	}
}

func TestEach(t *testing.T) {
	var e1, e2 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}

	s1 := New()
	s1.Add(e1)
	s1.Add(e2)

	found := make(map[string]bool)

	s1.Each(func(item string) bool {
		found[item] = true
		return true
	})

	if len(found) != 2 {
		t.Errorf("not all items traversed only %v", found)
	}

	found = make(map[string]bool)
	count := 0
	s1.Each(func(item string) bool {
		found[item] = true
		count++
		if count > 0 {
			return false
		}
		return true
	})

	if len(found) != 1 {
		t.Errorf("more than expected 1 items traversed %v", found)
	}
}

func TestIntersection(t *testing.T) {
	var e1, e2, e3 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}
	e = createRandomObject(e3)
	if v, ok := e.(string); ok {
		e3 = v
	}

	s1 := New()
	s1.Add(e1)
	s1.Add(e2)

	s2 := New()
	s2.Add(e2)
	s2.Add(e3)

	s3 := Intersection(s1, s2)

	if s3.Size() != 1 || !s3.Has(e2) {
		t.Errorf("expected the intersection to only contain '%v' but it is %v", e2, s3.List())
	}
}

func TestUnion(t *testing.T) {
	var e1, e2, e3 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}
	e = createRandomObject(e3)
	if v, ok := e.(string); ok {
		e3 = v
	}

	s1 := New()
	s1.Add(e1)
	s1.Add(e2)

	s2 := New()
	s2.Add(e2)
	s2.Add(e3)

	s3 := Union(s1, s2)

	if s3.Size() != 3 || !(s3.Has(e1) && s3.Has(e2) && s3.Has(e3)) {
		t.Errorf("expected the intersection to only contain %v but it is %v", e2, s3.List())
	}
}

func TestDifference(t *testing.T) {
	var e1, e2, e3 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}
	e = createRandomObject(e3)
	if v, ok := e.(string); ok {
		e3 = v
	}

	s1 := New()
	s1.Add(e1)
	s1.Add(e2)

	s2 := New()
	s2.Add(e2)
	s2.Add(e3)

	s3 := Difference(s1, s2)

	if s3.Size() != 1 || !s3.Has(e1) {
		t.Errorf("expected the intersection to only contain %v but it is %v", e2, s3.List())
	}
}

func TestSymmetricDifference(t *testing.T) {
	var e1, e2, e3 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	e = createRandomObject(e2)
	if v, ok := e.(string); ok {
		e2 = v
	}
	e = createRandomObject(e3)
	if v, ok := e.(string); ok {
		e3 = v
	}

	s1 := New()
	s1.Add(e1)
	s1.Add(e2)

	s2 := New()
	s2.Add(e2)
	s2.Add(e3)

	s3 := SymmetricDifference(s1, s2)

	if s3.Size() != 2 || !(s3.Has(e1) && s3.Has(e3)) {
		t.Errorf("expected the intersection to only contain %v but it is %v", e2, s3.List())
	}
}

func BenchmarkTypeSafeSetHasNonExisting(b *testing.B) {
	b.StopTimer()
	var e1 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	b.StartTimer()
	s := New()
	for i := 0; i < b.N; i++ {
		s.Has(e1)
	}
}

func BenchmarkInterfaceSetHasNonExisting(b *testing.B) {
	b.StopTimer()
	var e1 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	b.StartTimer()
	s := set.New(set.NonThreadSafe)
	for i := 0; i < b.N; i++ {
		s.Has(e1)
	}
}

func BenchmarkTypeSafeSetHasExisting(b *testing.B) {
	b.StopTimer()
	var e1 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	b.StartTimer()
	s := New()
	s.Add(e1)
	for i := 0; i < b.N; i++ {
		s.Has(e1)
	}
}

func BenchmarkInterfaceSetHasExisting(b *testing.B) {
	b.StopTimer()
	var e1 string
	e := createRandomObject(e1)
	if v, ok := e.(string); ok {
		e1 = v
	}
	b.StartTimer()
	s := set.New(set.NonThreadSafe)
	s.Add(e1)
	for i := 0; i < b.N; i++ {
		s.Has(e1)
	}
}

func BenchmarkTypeSafeSetHasExistingMany(b *testing.B) {
	s := New()
	b.StopTimer()
	var e1 string
	for i := 0; i < 10000; i++ {
		e := createRandomObject(e1)
		if v, ok := e.(string); ok {
			s.Add(v)
			if i == 5000 {
				e1 = v
			}
		}
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Has(e1)
	}
}

func BenchmarkInterfaceSetHasExistingMany(b *testing.B) {
	s := set.New(set.NonThreadSafe)
	b.StopTimer()
	var e1 string
	for i := 0; i < 10000; i++ {
		e := createRandomObject(e1)
		if v, ok := e.(string); ok {
			s.Add(v)
			if i == 5000 {
				e1 = v
			}
		}
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Has(e1)
	}
}

func BenchmarkTypeSafeSetAdd(b *testing.B) {
	b.StopTimer()
	var e string
	s := New()
	objs := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		e := createRandomObject(e)
		if v, ok := e.(string); ok {
			objs = append(objs, v)
		}
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Add(objs[i])
	}
}

func BenchmarkInterfaceSetAdd(b *testing.B) {
	b.StopTimer()
	var e string
	s := set.New(set.NonThreadSafe)
	objs := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		e := createRandomObject(e)
		if v, ok := e.(string); ok {
			objs = append(objs, v)
		}
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s.Add(objs[i])
	}
}

func createRandomObject(i interface{}) interface{} {
	v, ok := quick.Value(reflect.TypeOf(i), rand.New(rand.NewSource(time.Now().UnixNano())))
	if !ok {
		panic(fmt.Sprintf("unsupported type %v", i))
	}
	return v.Interface()
}
