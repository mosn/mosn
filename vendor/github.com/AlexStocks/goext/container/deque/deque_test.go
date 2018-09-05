package gxdeque

import (
	"container/list"
	"testing"
)

import (
	jjtesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

func TestPackage(t *testing.T) {
	gc.TestingT(t)
}

type suite struct {
	jjtesting.IsolationSuite
	q *Deque
}

var _ = gc.Suite(&suite{})

const testLen = 1000

func (s *suite) SetUpTest(c *gc.C) {
	s.q = New()
}

func (s *suite) TestInit(c *gc.C) {
	s.checkEmpty(c)
}

func (s *suite) TestStackBack(c *gc.C) {
	// Push many values on to the back.
	for i := 0; i < testLen; i++ {
		c.Assert(s.q.Len(), gc.Equals, i)
		s.q.PushBack(i)

		v, ok := s.q.PeekBack()
		c.Assert(ok, gc.Equals, true)
		c.Assert(v.(int), gc.Equals, i)
	}

	// Pop them all off from the back.
	for i := testLen; i > 0; i-- {
		c.Assert(s.q.Len(), gc.Equals, i)
		v, ok := s.q.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i-1)
	}

	s.checkEmpty(c)
}

func (s *suite) TestStackFront(c *gc.C) {
	// Push many values on to the front.
	for i := 0; i < testLen; i++ {
		c.Assert(s.q.Len(), gc.Equals, i)
		s.q.PushFront(i)

		v, ok := s.q.PeekFront()
		c.Assert(ok, gc.Equals, true)
		c.Assert(v.(int), gc.Equals, i)
	}

	// Pop them all off from the front.
	for i := testLen; i > 0; i-- {
		c.Assert(s.q.Len(), gc.Equals, i)
		v, ok := s.q.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i-1)
	}

	s.checkEmpty(c)
}

func (s *suite) TestQueueFromFront(c *gc.C) {
	// Push many values on to the back.
	for i := 0; i < testLen; i++ {
		s.q.PushBack(i)
	}

	// Pop them all off the front.
	for i := 0; i < testLen; i++ {
		v, ok := s.q.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i)
	}

	s.checkEmpty(c)
}

func (s *suite) TestQueueFromBack(c *gc.C) {
	// Push many values on to the front.
	for i := 0; i < testLen; i++ {
		s.q.PushFront(i)
	}

	// Pop them all off the back.
	for i := 0; i < testLen; i++ {
		v, ok := s.q.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(v.(int), gc.Equals, i)
	}

	s.checkEmpty(c)
}

func (s *suite) TestFrontBack(c *gc.C) {
	// Populate from the front and back.
	for i := 0; i < testLen; i++ {
		c.Assert(s.q.Len(), gc.Equals, i*2)
		s.q.PushFront(i)
		s.q.PushBack(i)
	}

	//  Remove half the items from the front and back.
	for i := testLen; i > testLen/2; i-- {
		c.Assert(s.q.Len(), gc.Equals, i*2)

		vb, ok := s.q.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vb.(int), gc.Equals, i-1)

		vf, ok := s.q.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vf.(int), gc.Equals, i-1)
	}

	// Expand out again.
	for i := testLen / 2; i < testLen; i++ {
		c.Assert(s.q.Len(), gc.Equals, i*2)
		s.q.PushFront(i)
		s.q.PushBack(i)
	}

	// Consume all.
	for i := testLen; i > 0; i-- {
		c.Assert(s.q.Len(), gc.Equals, i*2)

		vb, ok := s.q.PopBack()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vb.(int), gc.Equals, i-1)

		vf, ok := s.q.PopFront()
		c.Assert(ok, jc.IsTrue)
		c.Assert(vf.(int), gc.Equals, i-1)
	}

	s.checkEmpty(c)
}

func (s *suite) TestMaxLenFront(c *gc.C) {
	const max = 5
	d := NewWithMaxLen(max)

	// Exceed the maximum length by 2
	for i := 0; i < max+2; i++ {
		d.PushFront(i)
	}

	// Observe the the first 2 items on the back were dropped.
	v, ok := d.PopBack()
	c.Assert(ok, jc.IsTrue)
	c.Assert(v.(int), gc.Equals, 2)
}

func (s *suite) TestMaxLenBack(c *gc.C) {
	const max = 5
	d := NewWithMaxLen(max)

	// Exceed the maximum length by 3
	for i := 0; i < max+3; i++ {
		d.PushBack(i)
	}

	// Observe the the first 3 items on the front were dropped.
	v, ok := d.PopFront()
	c.Assert(ok, jc.IsTrue)
	c.Assert(v.(int), gc.Equals, 3)
}

func (s *suite) TestBlockAllocation(c *gc.C) {
	// This test confirms that the Deque allocates and deallocates
	// blocks as expected.

	for i := 0; i < testLen; i++ {
		s.q.PushFront(i)
		s.q.PushBack(i)
	}
	// 2000 items at a blockLen of 64:
	// 31 full blocks + 1 partial front + 1 partial back = 33
	c.Assert(s.q.blocks.Len(), gc.Equals, 33)

	for i := 0; i < testLen; i++ {
		s.q.PopFront()
		s.q.PopBack()
	}
	// At empty there should be just 1 block.
	c.Assert(s.q.blocks.Len(), gc.Equals, 1)
}

func (s *suite) checkEmpty(c *gc.C) {
	c.Assert(s.q.Len(), gc.Equals, 0)

	_, ok := s.q.PopFront()
	c.Assert(ok, jc.IsFalse)

	_, ok = s.q.PopBack()
	c.Assert(ok, jc.IsFalse)
}

func (s *suite) BenchmarkPushBackList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushBack(i)
	}
}

func (s *suite) BenchmarkPushBackDeque(c *gc.C) {
	d := New()
	for i := 0; i < c.N; i++ {
		d.PushBack(i)
	}
}

func (s *suite) BenchmarkPushFrontList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushFront(i)
	}
}

func (s *suite) BenchmarkPushFrontDeque(c *gc.C) {
	d := New()
	for i := 0; i < c.N; i++ {
		d.PushFront(i)
	}
}

func (s *suite) BenchmarkPushPopFrontList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushFront(i)
	}
	for i := 0; i < c.N; i++ {
		elem := l.Front()
		_ = elem.Value
		l.Remove(elem)
	}
}

func (s *suite) BenchmarkPushPopFrontDeque(c *gc.C) {
	d := New()
	for i := 0; i < c.N; i++ {
		d.PushFront(i)
	}
	for i := 0; i < c.N; i++ {
		_, _ = d.PopFront()
	}
}

func (s *suite) BenchmarkPushPopBackList(c *gc.C) {
	l := list.New()
	for i := 0; i < c.N; i++ {
		l.PushBack(i)
	}
	for i := 0; i < c.N; i++ {
		elem := l.Back()
		_ = elem.Value
		l.Remove(elem)
	}
}

func (s *suite) BenchmarkPushPopBackDeque(c *gc.C) {
	d := New()
	for i := 0; i < c.N; i++ {
		d.PushBack(i)
	}
	for i := 0; i < c.N; i++ {
		_, _ = d.PopBack()
	}
}
