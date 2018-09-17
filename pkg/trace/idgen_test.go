package trace

import "testing"

func TestIdGenerator_GenerateTraceId(t *testing.T) {
	traceId := IdGen().GenerateTraceId()
	if traceId == "" {
		t.Error("Generate traceId is empty")
	}
}

func TestIdGenerator_ipToHexString(t *testing.T) {
	ipHex := ipToHexString()

	if ipHex == "" {
		t.Error("IP to Hex is empty.")
	}

	if len(ipHex) != 8 {
		t.Error("Wrong format of IP to Hex, the length should be 8")
	}
}
