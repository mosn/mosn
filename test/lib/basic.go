package lib

import (
	"sync"
	"testing"
)

type testCase struct {
	setup    func() error
	teardown func()
	actions  []action
	verify   func() error
}

type action struct {
	desrcibe string
	do       func() error
}

func (c *testCase) Setup() error {
	if c.setup != nil {
		return c.setup()
	}
	return nil
}

func (c *testCase) Teardown() {
	if c.teardown != nil {
		c.teardown()
	}
}

var (
	globalMutex sync.Mutex
	globalCase  *testCase
)

func Scenario(t *testing.T, describe string, setScenario func()) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalCase = &testCase{}
	setScenario()
	defer globalCase.Teardown()
	t.Logf("%s setup", describe)
	if err := globalCase.Setup(); err != nil {
		t.Errorf("%s setup failed, error: %v", describe, err)
	}
	t.Logf("%s running", describe)
	for _, action := range globalCase.actions {
		t.Logf("doing: %s ", action.desrcibe)
		if err := action.do(); err != nil {
			t.Errorf("%s execute failed, error: %v", action.desrcibe, err)
		}
		t.Logf("finished: %s", action.desrcibe)
	}
	// verify is not required
	if globalCase.verify != nil {
		if err := globalCase.verify(); err != nil {
			t.Errorf("%s verify failed, error: %v", describe, err)
		}
	}
}

func Setup(setup func() error) {
	if globalCase.setup != nil {
		panic("scenario is already setted a setup")
	}
	globalCase.setup = setup
}

func TearDown(teardown func()) {
	if globalCase.teardown != nil {
		panic("scenario is already setted a teardown")
	}
	globalCase.teardown = teardown
}

func Execute(describe string, do func() error) {
	globalCase.actions = append(globalCase.actions, action{
		desrcibe: describe,
		do:       do,
	})
}

func Verify(do func() error) {
	globalCase.verify = do
}
