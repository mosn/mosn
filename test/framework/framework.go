package framework

import (
	"sync"
	"testing"
)

// scenario desrcibes a test scenario
type scenario struct {
	t        *testing.T
	setup    func()
	teardown func()
	cases    []testCase
}

func (s *scenario) Setup() {
	if s.setup != nil {
		s.setup()
	}
}

func (s *scenario) TearDown() {
	if s.teardown != nil {
		s.teardown()
	}
}

func (s *scenario) Run() {
	for _, c := range s.cases {
		s.t.Logf("run case: %s", c.desrcibe)
		c.do()
		s.t.Logf("finish case: %s", c.desrcibe)
	}
}

// testCase desrcibes a case in a scenario
type testCase struct {
	desrcibe string
	do       func()
}

// testContext wraps a scenario running
type testContext struct {
	// we use a mutext to lock text context makes the scenario is running by sequence
	mutex sync.Mutex
	// a test scenario
	scenario *scenario
	// isScenario desrcibes the function states
	isScenario bool
	// isInit desrcibes the functions states
	isInit bool
	// keeps *testing.T
	t *testing.T
}

func (ctx *testContext) valid() bool {
	return ctx.isScenario && ctx.scenario != nil
}

func (ctx *testContext) start(t *testing.T) {
	ctx.scenario = &scenario{t: t}
	ctx.t = t
	ctx.isScenario = true
}

func (ctx *testContext) clean() {
	if ctx.scenario == nil {
		return
	}
	ctx.scenario.TearDown()
	ctx.scenario = nil
	ctx.isScenario = false
	ctx.isInit = false
}

func (ctx *testContext) RunCase() {
	if ctx.scenario == nil {
		return
	}
	ctx.scenario.Run()
}

// a global test context
var context = &testContext{}

func Scenario(t *testing.T, describe string, initialize func()) {
	context.mutex.Lock()
	defer context.mutex.Unlock()
	defer context.clean()
	context.start(t)
	// init scenario
	initialize()
	context.isInit = true
	// executing scenario
	t.Logf("setup scenario: %s ", describe)
	context.scenario.Setup()
	t.Logf("run scenario: %s ", describe)
	context.RunCase()
}

func Setup(setup func()) {
	if !context.valid() {
		panic("setup should be called in scenario")
	}
	if context.scenario.setup != nil {
		panic("setup is already setted")
	}
	context.scenario.setup = setup
}

func TearDown(teardown func()) {
	if !context.valid() {
		panic("teardown should be called in scenario")
	}
	if context.scenario.teardown != nil {
		panic("teardown is already setted")
	}
	context.scenario.teardown = teardown
}

func Case(desrcibe string, do func()) {
	if !context.valid() {
		panic("case should be called in scenario")
	}
	context.scenario.cases = append(context.scenario.cases, testCase{
		desrcibe: desrcibe,
		do:       do,
	})
}

func Verify(actual interface{}, assert Assert, expected ...interface{}) {
	if !context.isInit {
		panic("verify should be called after initialized")
	}
	assert(context.t, actual, expected...)
}
