//golang实现set功能
//
//获取长度不可靠
package set

import (
	"sync"
	"sync/atomic"
)

type Set struct {
	el     sync.Map
	length int32
}

func NewSet(opt ...interface{}) *Set {
	s := new(Set)
	for _, o := range opt {
		s.Add(o)
	}
	return s
}

func (s *Set) Add(e interface{}) bool {
	_, ok := s.el.LoadOrStore(e, s.length)
	if !ok {
		atomic.AddInt32(&s.length, 1)
	}
	return ok
}

func (s *Set) Remove(e interface{}) {
	atomic.AddInt32(&s.length, -1)
	//没有元素时重置为0
	atomic.CompareAndSwapInt32(&s.length, -1, 0)
	s.el.Delete(e)
}

func (s *Set) Clear() {
	atomic.SwapInt32(&s.length, 0)
	var newEl sync.Map
	s.el = newEl
}

func (s *Set) Slice() (list []interface{}) {
	s.el.Range(func(k, v interface{}) bool {
		list = append(list, k)
		return true
	})
	return list
}

func (s *Set) Len() int32 {
	return s.length
}

func (s *Set) Has(e interface{}) bool {
	if _, ok := s.el.Load(e); ok {
		return true
	}
	return false
}
