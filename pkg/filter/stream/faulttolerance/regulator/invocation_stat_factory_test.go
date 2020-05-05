package regulator

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_InvocationFactory(t *testing.T) {
	//
	factory := &InvocationStatFactory{
		invocationStats: new(sync.Map),
		regulator:       NewMockRegulator(),
	}
	//
	dimension_1 := NewMockInvocationDimension("111", "AAA")
	dimension_2 := NewMockInvocationDimension("222", "BBB")
	dimension_3 := NewMockInvocationDimension("333", "CCC")

	go func() {
		for i := 0; i < 10; i++ {
			go func() {
				factory.GetInvocationStat(nil, dimension_1)
				factory.GetInvocationStat(nil, dimension_2)
				factory.GetInvocationStat(nil, dimension_3)
			}()
		}
	}()
	go func() {
		for i := 0; i < 10; i++ {
			go func() {
				factory.GetInvocationStat(nil, dimension_1)
				factory.GetInvocationStat(nil, dimension_2)
				factory.GetInvocationStat(nil, dimension_3)
			}()
		}
	}()
	go func() {
		for i := 0; i < 10; i++ {
			go func() {
				factory.GetInvocationStat(nil, dimension_1)
				factory.GetInvocationStat(nil, dimension_2)
				factory.GetInvocationStat(nil, dimension_3)
			}()
		}
	}()

	time.Sleep(3 * time.Second)

	regulator := factory.regulator.(*MockRegulator)
	if regulator.count != 3 {
		t.Error("Test_InvocationFactory Failed")
	}
	if value, ok := regulator.source.Load("111"); ok {
		stat := value.(*InvocationStat)
		if stat.GetInvocationKey() != "111" {
			t.Error("Test_InvocationFactory Failed")
		}
	} else {
		t.Error("Test_InvocationFactory Failed")
	}
	if value, ok := regulator.source.Load("222"); ok {
		stat := value.(*InvocationStat)
		if stat.GetInvocationKey() != "222" {
			t.Error("Test_InvocationFactory Failed")
		}
	} else {
		t.Error("Test_InvocationFactory Failed")
	}
	if value, ok := regulator.source.Load("333"); ok {
		stat := value.(*InvocationStat)
		if stat.GetInvocationKey() != "333" {
			t.Error("Test_InvocationFactory Failed")
		}
	} else {
		t.Error("Test_InvocationFactory Failed")
	}
}

type MockRegulator struct {
	count  int32
	source *sync.Map
}

func NewMockRegulator() *MockRegulator {
	return &MockRegulator{
		count:  0,
		source: new(sync.Map),
	}
}

func (r *MockRegulator) Regulate(stat *InvocationStat) {
	atomic.AddInt32(&r.count, 1)
	r.source.Store(stat.GetInvocationKey(), stat)
}

func (r *MockRegulator) GetCount() int32 {
	return r.count
}

func (r *MockRegulator) GetSource() *sync.Map {
	return r.source
}
