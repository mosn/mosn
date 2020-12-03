package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// simple example for test
func TestFrameworkExample(t *testing.T) {
	Scenario(t, "test example without setup and teardown", func() {
		a := 0
		Case("add with verify", func() {
			a = a + 1
			Verify(a, Equal, 1)
		})
		Case("add without verify", func() {
			a = a + 1
		})
		Case("only verify", func() {
			Verify(a, Equal, 2)
		})
	})
}

func TestFrameworkExample2(t *testing.T) {
	a := 0 // not recommended
	Scenario(t, "test example with setup and teardown", func() {
		Setup(func() {
			a = 100
		})
		TearDown(func() {
			a = 0
		})
		Case("double", func() {
			a = a * 2
			Verify(a, Equal, 200)
		})
	})
	// Scenario execute by sequence
	Scenario(t, "test example by sequence", func() {
		Case("verify teardown", func() {
			Verify(a, Equal, 0)
		})
	})
}

// Invalid Test
func TestInvalidTest(t *testing.T) {
	for _, f := range []func(){
		func() {
			Setup(func() {
			})
		},
		func() {
			TearDown(func() {
			})
		},
		func() {
			Case("panic", func() {
				t.Log("test")
			})
		},

		func() {
			Verify(1, Equal, 1)
		},
		func() {
			Scenario(t, "panic", func() {
				Verify(1, Equal, 1) // Verify should be called in Case
			})
		},
	} {
		assert.Panics(t, f)
	}
}
