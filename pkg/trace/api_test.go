package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/types"
)

func TestAPI(t *testing.T) {
	driver := NewDefaultDriverImpl()
	proto := types.ProtocolName("test")

	driver.Register(proto, func(config map[string]interface{}) (types.Tracer, error) {
		return &mockTracer{}, nil
	})

	driver.Init(nil)
	RegisterDriver("driverdriver", driver)

	err := Init("driverdriver", map[string]interface{}{})
	assert.Nil(t, err)
}

func TestEnable(t *testing.T) {
	Enable()
	assert.True(t, IsEnabled())

	Disable()
	assert.False(t, IsEnabled())
}
