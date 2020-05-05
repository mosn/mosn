package regulator

import (
	"testing"
)

func Test_Stat(t *testing.T) {
	//
	dimension := NewMockInvocationDimension("111", "ooo")
	stat := NewInvocationStat(nil, dimension)
	//
	stat.Call(false)
	stat.Call(false)
	stat.Call(true)
	stat.Call(false)
	stat.Call(true)
	stat.Call(false)
	if ok, exceptionRate := stat.GetExceptionRate(); !ok || exceptionRate != 0.33 {
		t.Error("Test_Stat Failed")
	}
	//
	snapshot := stat.Snapshot()
	if call, exception := snapshot.GetCount(); call != 6 || exception != 2 {
		t.Error("Test_Stat Failed")
	}
	stat.Call(false)
	stat.Call(true)
	stat.Call(false)
	if call, exception := stat.GetCount(); call != 9 || exception != 3 {
		t.Error("Test_Stat Failed")
	}
	if call, exception := snapshot.GetCount(); call != 6 || exception != 2 {
		t.Error("Test_Stat Failed")
	}
	//
	stat.Update(snapshot)
	if call, exception := stat.GetCount(); call != 3 || exception != 1 {
		t.Error("Test_Stat Failed")
	}
}
