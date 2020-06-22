//Package set implement
package set

import (
	"sync"
	"sync/atomic"
)

//Set struct
type Set struct {
	el     sync.Map
	length int32
}

//NewSet news a set
func NewSet(opt ...interface{}) *Set {
	s := new(Set)
	for _, o := range opt {
		s.Add(o)
	}
	return s
}

//Add add an element to Set
func (s *Set) Add(e interface{}) bool {
	_, ok := s.el.LoadOrStore(e, s.length)
	if !ok {
		atomic.AddInt32(&s.length, 1)
	}
	return ok
}

//Remove remove an element from set.
func (s *Set) Remove(e interface{}) {
	atomic.AddInt32(&s.length, -1)
	//没有元素时重置为0
	atomic.CompareAndSwapInt32(&s.length, -1, 0)
	s.el.Delete(e)
}

//Clear clear the set.
func (s *Set) Clear() {
	atomic.SwapInt32(&s.length, 0)
	s.el.Range(func(key interface{}, value interface{}) bool {
		s.el.Delete(key)
		return true
	})
}

//Slice init set from []interface.
func (s *Set) Slice() (list []interface{}) {
	s.el.Range(func(k, v interface{}) bool {
		list = append(list, k)
		return true
	})
	return list
}

//Len return the length of the set
func (s *Set) Len() int32 {
	return s.length
}

//Has indicates whether the set has the element.
func (s *Set) Has(e interface{}) bool {
	if _, ok := s.el.Load(e); ok {
		return true
	}
	return false
}
