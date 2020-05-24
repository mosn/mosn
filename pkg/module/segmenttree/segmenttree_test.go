package segmenttree

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_segmentTree(t *testing.T) {
	ns := []Node{
		{
			Value:      1,
			RangeStart: 0,
			RangeEnd:   1,
		},
		{
			Value:      2,
			RangeStart: 1,
			RangeEnd:   2,
		},
		{
			Value:      3,
			RangeStart: 2,
			RangeEnd:   3,
		},
	}

	f := func(l, r interface{}) interface{} {
		return l.(int) + r.(int)
	}

	tree := NewTree(ns, f)
	if !reflect.DeepEqual(tree.data, []interface{}{
		nil, 6, 5, 1, 2, 3,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeStart, map[int]uint64{
		1: 0,
		2: 1,
		3: 0,
		4: 1,
		5: 2,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeEnd, map[int]uint64{
		1: 3,
		2: 3,
		3: 1,
		4: 2,
		5: 3,
	}) {
		t.FailNow()
	}

	ns = []Node{
		{
			Value:      1,
			RangeStart: 0,
			RangeEnd:   1,
		},
		{
			Value:      2,
			RangeStart: 1,
			RangeEnd:   2,
		},
		{
			Value:      3,
			RangeStart: 2,
			RangeEnd:   3,
		},
		{
			Value:      4,
			RangeStart: 3,
			RangeEnd:   4,
		},
	}

	tree = NewTree(ns, f)
	if !reflect.DeepEqual(tree.data, []interface{}{
		nil, 10, 3, 7, 1, 2, 3, 4,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeStart, map[int]uint64{
		1: 0,
		2: 0,
		3: 2,
		4: 0,
		5: 1,
		6: 2,
		7: 3,
	}) {
		t.FailNow()
	}
	if !reflect.DeepEqual(tree.rangeEnd, map[int]uint64{
		1: 4,
		2: 2,
		3: 4,
		4: 1,
		5: 2,
		6: 3,
		7: 4,
	}) {
		t.FailNow()
	}
}

func Test_updateTree(t *testing.T) {
	ns := []Node{
		{
			Value:      1,
			RangeStart: 0,
			RangeEnd:   1,
		},
		{
			Value:      2,
			RangeStart: 1,
			RangeEnd:   2,
		},
		{
			Value:      3,
			RangeStart: 2,
			RangeEnd:   3,
		},
		{
			Value:      4,
			RangeStart: 3,
			RangeEnd:   4,
		},
	}

	f := func(l, r interface{}) interface{} {
		return l.(int) + r.(int)
	}

	tree := NewTree(ns, f)
	leaf, err  := tree.Leaf(3)
	if err != nil {
		t.Error(err)
	}

	if !assert.Equalf(t, 7, leaf.index, "leaf index should be 7")  {
		t.FailNow()
	}

	leaf.Value = 10
	tree.Update(leaf)

	if !reflect.DeepEqual(tree.data, []interface{}{
		nil, 16, 3, 13, 1, 2, 3, 10,
	}) {
		t.FailNow()
	}
}
