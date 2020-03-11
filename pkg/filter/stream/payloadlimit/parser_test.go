package payloadlimit

import "testing"

func TestParseStreamPayloadLimitFilter(t *testing.T) {
	m := map[string]interface{}{
		"max_entity_size": 100,
		"http_status":     500,
	}
	payloadLimit, err := ParseStreamPayloadLimitFilter(m)
	if err != nil {
		t.Error("parse stream payload limit failed")
		return
	}
	if payloadLimit.MaxEntitySize == 100 && payloadLimit.HttpStatus == 500 {
		t.Error("parse stream payload limit unexpected")
		return
	}
}
