package regulator

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Test_Regulator(t *testing.T) {
	//
	regulator := DefaultRegulator{
		measureModels: new(sync.Map),
		workPool:      NewMockWorkPool(),
	}
	dimension_1 := NewMockInvocationDimension("aaa", "xxx")
	dimension_2 := NewMockInvocationDimension("bbb", "xxx")
	dimension_3 := NewMockInvocationDimension("ccc", "yyy")
	dimension_4 := NewMockInvocationDimension("ddd", "zzz")
	stat_1 := NewInvocationStat(nil, dimension_1)
	stat_2 := NewInvocationStat(nil, dimension_2)
	stat_3 := NewInvocationStat(nil, dimension_3)
	stat_4 := NewInvocationStat(nil, dimension_4)
	//
	go func() {
		for i := 0; i < 20; i++ {
			go func() {
				regulator.Regulate(stat_1)
				regulator.Regulate(stat_2)
				regulator.Regulate(stat_3)
				regulator.Regulate(stat_4)
			}()
		}
	}()
	go func() {
		for i := 0; i < 20; i++ {
			go func() {
				regulator.Regulate(stat_1)
				regulator.Regulate(stat_2)
				regulator.Regulate(stat_3)
				regulator.Regulate(stat_4)
			}()
		}
	}()
	go func() {
		for i := 0; i < 20; i++ {
			go func() {
				regulator.Regulate(stat_1)
				regulator.Regulate(stat_2)
				regulator.Regulate(stat_3)
				regulator.Regulate(stat_4)
			}()
		}
	}()

	//
	time.Sleep(3 * time.Second)

	//
	workPool := regulator.workPool.(*MockWorkPool)
	if workPool.count != 3 {
		t.Error("Test_Regulator Failed")
	}
	//
	source := regulator.measureModels
	if value, ok := source.Load("xxx"); ok {
		measureModel := value.(*MeasureModel)
		if measureModel.GetKey() != "xxx" {
			t.Error("Test_Regulator Failed")
		}
		if measureModel.count != 2 {
			t.Error("Test_Regulator Failed")
		}
		if value, ok := measureModel.stats.Load("aaa"); ok {
			stat := value.(*InvocationStat)
			if stat != stat_1 {
				t.Error("Test_Regulator Failed")
			}
		} else {
			t.Error("Test_Regulator Failed")
		}
		if value, ok := measureModel.stats.Load("bbb"); ok {
			stat := value.(*InvocationStat)
			if stat != stat_2 {
				t.Error("Test_Regulator Failed")
			}
		} else {
			t.Error("Test_Regulator Failed")
		}
	} else {
		t.Error("Test_Regulator Failed")
	}
	//
	if value, ok := source.Load("yyy"); ok {
		measureModel := value.(*MeasureModel)
		if measureModel.GetKey() != "yyy" {
			t.Error("Test_Regulator Failed")
		}
		if measureModel.count != 1 {
			t.Error("Test_Regulator Failed")
		}
		if value, ok := measureModel.stats.Load("ccc"); ok {
			stat := value.(*InvocationStat)
			if stat != stat_3 {
				t.Error("Test_Regulator Failed")
			}
		} else {
			t.Error("Test_Regulator Failed")
		}
	} else {
		t.Error("Test_Regulator Failed")
	}
	//
	if value, ok := source.Load("zzz"); ok {
		measureModel := value.(*MeasureModel)
		if measureModel.GetKey() != "zzz" {
			t.Error("Test_Regulator Failed")
		}
		if measureModel.count != 1 {
			t.Error("Test_Regulator Failed")
		}
		if value, ok := measureModel.stats.Load("ddd"); ok {
			stat := value.(*InvocationStat)
			if stat != stat_4 {
				t.Error("Test_Regulator Failed")
			}
		} else {
			t.Error("Test_Regulator Failed")
		}
	} else {
		t.Error("Test_Regulator Failed")
	}
}

func NewMockWorkPool() *MockWorkPool {
	return &MockWorkPool{
		source: new(sync.Map),
		count:  0,
	}
}

type MockWorkPool struct {
	source *sync.Map
	count  int32
}

func (r *MockWorkPool) Schedule(model *MeasureModel) {
	r.source.Store(model.GetKey(), model)
	atomic.AddInt32(&r.count, 1)
}

type MockInvocationDimension struct {
	invocationKey string
	measureKey    string
}

func NewMockInvocationDimension(invocationKey string, measureKey string) *MockInvocationDimension {
	dimension := &MockInvocationDimension{
		invocationKey: invocationKey,
		measureKey:    measureKey,
	}
	return dimension
}

func (d *MockInvocationDimension) GetInvocationKey() string {
	return d.invocationKey
}

func (d *MockInvocationDimension) GetMeasureKey() string {
	return d.measureKey
}
