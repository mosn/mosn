package v2

import "testing"

func Test_Create_1(t *testing.T) {
	address := "127.0.0.1"
	length := uint32(32)
	ipRange := Create(address, length)
	if !(ipRange.Length == length && ipRange.Address == address) {
		t.Error("create ip 1 range not match")
	}
}

func Test_Create_2(t *testing.T) {
	address := "127.0.0.1"
	length := uint32(24)
	ipRange := Create(address, length)
	if !(ipRange.Length == length && ipRange.Address == "127.0.0.0") {
		t.Error("create ip 2 range not match")
	}
}

func Test_Create_3(t *testing.T) {
	address := "127.0.0.1"
	length := uint32(0)
	ipRange := Create(address, length)
	if !(ipRange.Length == 0 && ipRange.Address == "0.0.0.0") {
		t.Error("create ip 3 range not match")
	}
}

func Test_IsInRange(t *testing.T) {
	address := "192.168.0.1"
	length := uint32(24)
	ipRange := Create(address, length)
	if !(ipRange.Length == length && ipRange.Address == "192.168.0.0") {
		t.Error("create ip range not match")
	}
	if !ipRange.IsInRange("192.168.0.128") {
		t.Error("test ip range fail")
	}
	if ipRange.IsInRange("192.168.1.128") {
		t.Error("test ip range fail")
	}
}
