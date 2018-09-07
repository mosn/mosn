// Copyright 2015 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

package gxxorlist

import "testing"

func checkListLen(t *testing.T, l *XorList, len int) bool {
	if n := l.Len(); n != len {
		t.Errorf("l.Len() = %d, want %d", n, len)
		return false
	}
	return true
}

func checkListPointers(t *testing.T, l *XorList, es []*XorElement) {
	if !checkListLen(t, l, len(es)) {
		return
	}

	// zero length lists must be the zero value or properly initialized (sentinel circle)
	if len(es) == 0 {
		if ptr(l.head.PN) != &l.tail {
			t.Errorf("l.head.PN = %x, &l.tail = %p; both should both be equal", l.head.PN, l.tail)
		}
		if ptr(l.tail.PN) != &l.head {
			t.Errorf("l.tail.PN = %x, &l.head = %p; both should both be equal", l.tail.PN, l.head)
		}

		return
	}
	// len(es) > 0

	i := 0
	var prev *XorElement
	for e, p := l.Front(); e != nil; e, p = e.Next(p), e {
		if e != es[i] {
			t.Errorf("elt[%d] = %p, want %p", i, es[i], e)
		}

		prev = &l.head
		if 0 < i {
			prev = es[i-1]
		}
		if p != prev {
			t.Errorf("elt[%d](%p).prev = %p, want %p", i, e, p, prev)
		}

		i++
	}

	i = len(es) - 1
	var next *XorElement
	for e, n := l.Back(); e != nil; e, n = e.Prev(n), e {
		if e != es[i] {
			t.Errorf("elt[%d] = %p, want %p", i, es[i], e)
		}

		next = &l.tail
		if i < len(es)-1 {
			next = es[i+1]
		}
		if n != next {
			t.Errorf("elt[%d](%p).next = %p, want %p", i, e, n, next)
		}

		i--
	}
}

func TestList(t *testing.T) {
	l := New()
	checkListPointers(t, l, []*XorElement{})

	// Single element list
	e := l.PushFront("a")
	checkListPointers(t, l, []*XorElement{e})
	l.MoveToFront(e)
	checkListPointers(t, l, []*XorElement{e})
	l.MoveToBack(e)
	checkListPointers(t, l, []*XorElement{e})
	l.Remove(e)
	checkListPointers(t, l, []*XorElement{})

	// Bigger list
	e2 := l.PushFront(2)
	e1 := l.PushFront(1)
	e3 := l.PushBack(3)
	e4 := l.PushBack("banana")
	checkListPointers(t, l, []*XorElement{e1, e2, e3, e4})

	l.Remove(e2)
	checkListPointers(t, l, []*XorElement{e1, e3, e4})

	l.MoveToFront(e3) // move from middle
	checkListPointers(t, l, []*XorElement{e3, e1, e4})

	l.MoveToFront(e1)
	l.MoveToBack(e3) // move from middle
	checkListPointers(t, l, []*XorElement{e1, e4, e3})

	l.MoveToFront(e3) // move from back
	checkListPointers(t, l, []*XorElement{e3, e1, e4})
	l.MoveToFront(e3) // should be no-op
	checkListPointers(t, l, []*XorElement{e3, e1, e4})

	l.MoveToBack(e3) // move from front
	checkListPointers(t, l, []*XorElement{e1, e4, e3})
	l.MoveToBack(e3) // should be no-op
	checkListPointers(t, l, []*XorElement{e1, e4, e3})

	e2 = l.InsertBefore(2, e1) // insert before front
	checkListPointers(t, l, []*XorElement{e2, e1, e4, e3})
	l.Remove(e2)
	e2 = l.InsertBefore(2, e4) // insert before middle
	checkListPointers(t, l, []*XorElement{e1, e2, e4, e3})
	l.Remove(e2)
	e2 = l.InsertBefore(2, e3) // insert before back
	checkListPointers(t, l, []*XorElement{e1, e4, e2, e3})
	l.Remove(e2)

	e2 = l.InsertAfter(2, e1) // insert after front
	checkListPointers(t, l, []*XorElement{e1, e2, e4, e3})
	l.Remove(e2)
	e2 = l.InsertAfter(2, e4) // insert after middle
	checkListPointers(t, l, []*XorElement{e1, e4, e2, e3})
	l.Remove(e2)
	e2 = l.InsertAfter(2, e3) // insert after back
	checkListPointers(t, l, []*XorElement{e1, e4, e3, e2})
	l.Remove(e2)

	// Check standard iteration.
	sum := 0
	for e, p := l.Front(); e != nil; e, p = e.Next(p), e {
		if i, ok := e.Value.(int); ok {
			sum += i
		}
	}
	if sum != 4 {
		t.Errorf("sum over l = %d, want 4", sum)
	}

	// Clear all elements by iterating
	var next *XorElement
	for e, p := l.Front(); e != nil; e = next {
		next = e.Next(p)
		l.Remove(e)
	}
	checkListPointers(t, l, []*XorElement{})
}

func checkList(t *testing.T, l *XorList, es []interface{}) {
	if !checkListLen(t, l, len(es)) {
		return
	}

	i := 0
	for e, p := l.Front(); e != nil; e, p = e.Next(p), e {
		le := e.Value.(int)
		if le != es[i] {
			t.Errorf("elt[%d].Value = %v, want %v", i, le, es[i])
		}
		i++
	}
}

func TestExtending(t *testing.T) {
	l1 := New()
	l2 := New()

	l1.PushBack(1)
	l1.PushBack(2)
	l1.PushBack(3)

	l2.PushBack(4)
	l2.PushBack(5)

	l3 := New()
	l3.PushBackList(l1)
	checkList(t, l3, []interface{}{1, 2, 3})
	l3.PushBackList(l2)
	checkList(t, l3, []interface{}{1, 2, 3, 4, 5})

	l3 = New()
	l3.PushFrontList(l2)
	checkList(t, l3, []interface{}{4, 5})
	l3.PushFrontList(l1)
	checkList(t, l3, []interface{}{1, 2, 3, 4, 5})

	checkList(t, l1, []interface{}{1, 2, 3})
	checkList(t, l2, []interface{}{4, 5})

	l3 = New()
	l3.PushBackList(l1)
	checkList(t, l3, []interface{}{1, 2, 3})
	l3.PushBackList(l3)
	checkList(t, l3, []interface{}{1, 2, 3, 1, 2, 3})
	l3 = New()
	l3.PushFrontList(l1)
	checkList(t, l3, []interface{}{1, 2, 3})
	l3.PushFrontList(l3)
	checkList(t, l3, []interface{}{1, 2, 3, 1, 2, 3})

	l3 = New()
	l1.PushBackList(l3)
	checkList(t, l1, []interface{}{1, 2, 3})
	l1.PushFrontList(l3)
	checkList(t, l1, []interface{}{1, 2, 3})
}

func TestRemove(t *testing.T) {
	l := New()
	e1 := l.PushBack(1)
	e2 := l.PushBack(2)
	checkListPointers(t, l, []*XorElement{e1, e2})
	e, _ := l.Front()
	l.Remove(e)
	checkListPointers(t, l, []*XorElement{e2})
	l.Remove(e)
	checkListPointers(t, l, []*XorElement{e2})
}

func TestIssue4103(t *testing.T) {
	l1 := New()
	l1.PushBack(1)
	l1.PushBack(2)

	l2 := New()
	l2.PushBack(3)
	l2.PushBack(4)

	e, _ := l1.Front()
	l2.Remove(e) // l2 should not change because e is not an element of l2
	if n := l2.Len(); n != 2 {
		t.Errorf("l2.Len() = %d, want 2", n)
	}

	l1.InsertBefore(8, e)
	if n := l1.Len(); n != 3 {
		t.Errorf("l1.Len() = %d, want 3", n)
	}
}

func TestIssue6349(t *testing.T) {
	l := New()
	l.PushBack(1)
	l.PushBack(2)

	e, p := l.Front()
	l.Remove(e)
	if e.Value != 1 {
		t.Errorf("e.value = %d, want 1", e.Value)
	}
	if e.Next(p) != nil {
		t.Errorf("e.Next() != nil")
	}
	if e.Prev(p) != nil {
		t.Errorf("e.Prev() != nil")
	}
}

func TestMove(t *testing.T) {
	l := New()
	e1 := l.PushBack(1)
	e2 := l.PushBack(2)
	e3 := l.PushBack(3)
	e4 := l.PushBack(4)

	l.MoveAfter(e3, e3)
	checkListPointers(t, l, []*XorElement{e1, e2, e3, e4})
	l.MoveBefore(e2, e2)
	checkListPointers(t, l, []*XorElement{e1, e2, e3, e4})

	l.MoveAfter(e3, e2)
	checkListPointers(t, l, []*XorElement{e1, e2, e3, e4})
	l.MoveBefore(e2, e3)
	checkListPointers(t, l, []*XorElement{e1, e2, e3, e4})

	l.MoveBefore(e2, e4)
	checkListPointers(t, l, []*XorElement{e1, e3, e2, e4})
	e1, e2, e3, e4 = e1, e3, e2, e4

	l.MoveBefore(e4, e1)
	checkListPointers(t, l, []*XorElement{e4, e1, e2, e3})
	e1, e2, e3, e4 = e4, e1, e2, e3

	l.MoveAfter(e4, e1)
	checkListPointers(t, l, []*XorElement{e1, e4, e2, e3})
	e1, e2, e3, e4 = e1, e4, e2, e3

	l.MoveAfter(e2, e3)
	checkListPointers(t, l, []*XorElement{e1, e3, e2, e4})
	e1, e2, e3, e4 = e1, e3, e2, e4
}

// Test PushFront, PushBack, PushFrontList, PushBackList with uninitialized XorList
func TestZeroList(t *testing.T) {
	var l1 = new(XorList)
	l1.PushFront(1)
	checkList(t, l1, []interface{}{1})

	var l2 = new(XorList)
	l2.PushBack(1)
	checkList(t, l2, []interface{}{1})

	var l3 = new(XorList)
	l3.PushFrontList(l1)
	checkList(t, l3, []interface{}{1})

	var l4 = new(XorList)
	l4.PushBackList(l2)
	checkList(t, l4, []interface{}{1})
}

// Test that a list l is not modified when calling InsertBefore with a mark that is not an element of l.
func TestInsertBeforeUnknownMark(t *testing.T) {
	var l XorList
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)
	l.InsertBefore(1, new(XorElement))
	checkList(t, &l, []interface{}{1, 2, 3})
}

// Test that a list l is not modified when calling InsertAfter with a mark that is not an element of l.
func TestInsertAfterUnknownMark(t *testing.T) {
	var l XorList
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)
	l.InsertAfter(1, new(XorElement))
	checkList(t, &l, []interface{}{1, 2, 3})
}

// Test that a list l is not modified when calling MoveAfter or MoveBefore with a mark that is not an element of l.
func TestMoveUnkownMark(t *testing.T) {
	var l1 XorList
	e1 := l1.PushBack(1)
	checkList(t, &l1, []interface{}{1})

	var l2 XorList
	e2 := l2.PushBack(2)

	l1.MoveAfter(e1, e2)
	checkList(t, &l1, []interface{}{1})
	checkList(t, &l2, []interface{}{2})

	l1.MoveBefore(e1, e2)
	checkList(t, &l1, []interface{}{1})
	checkList(t, &l2, []interface{}{2})
}

func TestLoopRemove(t *testing.T) {
	l := New()
	checkListPointers(t, l, []*XorElement{})

	// build list
	e1 := l.PushBack(2)
	e2 := l.PushBack(1)
	e3 := l.PushBack(3)
	e4 := l.PushBack(2)
	e5 := l.PushBack(5)
	e6 := l.PushBack(2)
	checkListPointers(t, l, []*XorElement{e1, e2, e3, e4, e5, e6})
	for e, p := l.Front(); e != nil; e, p = e.Next(p), e {
		if e.Value.(int) == 2 {
			elem := e
			e, p = p, p.Prev(e)
			l.Remove(elem)
		}
	}
	checkListPointers(t, l, []*XorElement{e2, e3, e5})
}
