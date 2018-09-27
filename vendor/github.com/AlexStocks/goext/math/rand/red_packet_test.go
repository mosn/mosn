package gxrand

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestCalRedPacket(t *testing.T) {
	packet, err := CalRedPacket(10, 10)
	if assert.Empty(t, err) {
		t.Logf("red packats:%+v\n", packet)
	} else {
		t.Errorf("CalRedPacket(10, 10) = error{%s}", err)
	}
}
