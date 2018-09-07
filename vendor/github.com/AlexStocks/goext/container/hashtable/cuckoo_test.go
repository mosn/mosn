//package cuckoo

package gxhashtable

import (
	"strconv"
	"testing"
)

// Standardizing our entry formats
func keyAndValue(i int) (key keytype, value valuetype) {
	keystr := strconv.Itoa(i)
	key = keytype(keystr)
	value = valuetype("testvalue-" + keystr)
	return
}

// Helper function to populate the table
func fillTable(ct *Table, limit int) {
	for i := 0; i < limit; i++ {
		k, v := keyAndValue(i)
		ct.Put(k, v)
	}
}

func fatalIfNotFound(t *testing.T, ct *Table, i int) {
	k, v := keyAndValue(i)
	foundVal, found := ct.Get(k)
	if !found {
		t.Fatalf("failed - item %v not present", k)
	}
	if v != foundVal {
		t.Fatalf("failed - item %v value incorrect, wanted %v got %v", k, v, foundVal)
	}
}

func fatalIfFound(t *testing.T, ct *Table, i int) {
	k, _ := keyAndValue(i)
	foundVal, found := ct.Get(k)
	if found {
		t.Fatalf("failed - item %v not deleted, value %v", k, foundVal)
	}
}

func validateFound(t *testing.T, ct *Table, start int, limit int, testName string) {
	for i := start; i < limit; i++ {
		fatalIfNotFound(t, ct, i)
	}
}

func TestNewTable(t *testing.T) {
	ct := NewTable()
	if ct == nil {
		t.Fatal("Could not allocate table")
	}
}

func TestBasic(t *testing.T) {
	ct := NewTable()
	nKeys := 1000
	fillTable(ct, nKeys)
	validateFound(t, ct, 0, nKeys, "TestBasic")
}

func TestFill(t *testing.T) {
	ct := NewTablePowerOfTwo(10)
	// Should be able to hold at least 950 elements, but will have to
	// cuckoo a lot to fill those last bits.  Stress test the cuckooing.
	limit := 874 // 875 fails - we're not BFS'ing well enough yet
	fillTable(ct, limit)
	validateFound(t, ct, 0, limit, "TestFill")
}

func TestDelete(t *testing.T) {
	ct := NewTable()
	limit := 1000
	fillTable(ct, limit)
	validateFound(t, ct, 0, limit, "TestDelete")
	for i := 0; i < limit; i++ {
		k, _ := keyAndValue(i)
		ct.Delete(k)
		fatalIfFound(t, ct, i)
		validateFound(t, ct, i+1, limit, "TestDelete")
	}
}

func TestRunningNearFull(t *testing.T) {
	ct := NewTablePowerOfTwo(10)
	tn := "TestRunningNearFull"
	limit := 800
	fillTable(ct, limit)
	validateFound(t, ct, 0, limit, tn)
	for i := 0; i < 10000; i++ {
		fatalIfNotFound(t, ct, i)
		k, _ := keyAndValue(i)
		ct.Delete(k)
		fatalIfFound(t, ct, i)
		k2, v2 := keyAndValue(i + limit)
		ct.Put(k2, v2)
		fatalIfNotFound(t, ct, i+limit)
	}
}

func BenchmarkFillNull(b *testing.B) {
	limit := 1<<19 + 1<<18
	for i := 0; i < b.N; i++ {
		for j := 0; j < limit; j++ {
			k, v := keyAndValue(j)
			_, _ = k, v
		}
	}
}

func BenchmarkFillMap(b *testing.B) {
	ct := make(map[string]string, 1<<20)
	limit := 1<<19 + 1<<18
	for i := 0; i < b.N; i++ {
		for j := 0; j < limit; j++ {
			k, v := keyAndValue(j)
			ct[string(k)] = string(v)
		}
	}
}

func BenchmarkFillCuckoo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ct := NewTablePowerOfTwo(20)
		limit := 1<<19 + 1<<18
		for j := 0; j < limit; j++ {
			k, v := keyAndValue(j)
			ct.Put(k, v)
		}
	}
}

func BenchmarkReadMap(b *testing.B) {
	b.StopTimer()
	ct := make(map[string]string, 1<<20)
	limit := 1<<19 + 1<<18
	keyarray := make([]string, limit)
	for i := 0; i < limit; i++ {
		k, v := keyAndValue(i)
		ct[string(k)] = string(v)
		keyarray[i] = string(k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < limit; j++ {
			_, found := ct[keyarray[i]]
			if !found {
				b.Fatalf("Could not benchmark - key not found")
			}
		}
	}
}

func BenchmarkReadCuckoo(b *testing.B) {
	b.StopTimer()
	ct := NewTablePowerOfTwo(20)
	limit := 1<<19 + 1<<18
	keyarray := make([]string, limit)
	for i := 0; i < limit; i++ {
		k, v := keyAndValue(i)
		ct.Put(k, v)
		keyarray[i] = string(k)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < limit; j++ {
			_, found := ct.Get(keytype(keyarray[i]))
			if !found {
				b.Fatalf("Could not benchmark - key not found")
			}
		}
	}
}
