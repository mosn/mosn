// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interpreter

import (
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/interpreter/functions"
)

// Interpretable can accept a given Activation and produce a value along with
// an accompanying EvalState which can be used to inspect whether additional
// data might be necessary to complete the evaluation.
type Interpretable interface {
	// ID value corresponding to the expression node.
	ID() int64

	// Eval an Activation to produce an output.
	Eval(activation Activation) ref.Val
}

// Core Interpretable implementations used during the program planning phase.

type evalTestOnly struct {
	id        int64
	op        Interpretable
	field     types.String
	fieldType *ref.FieldType
}

// ID implements the Interpretable interface method.
func (test *evalTestOnly) ID() int64 {
	return test.id
}

// Eval implements the Interpretable interface method.
func (test *evalTestOnly) Eval(ctx Activation) ref.Val {
	// Handle field selection on a proto in the most efficient way possible.
	if test.fieldType != nil {
		opAttr, ok := test.op.(instAttr)
		if ok {
			opVal, err := opAttr.Attr().Resolve(ctx)
			if err != nil {
				return types.NewErr(err.Error())
			}
			refVal, ok := opVal.(ref.Val)
			if ok {
				opVal = refVal.Value()
			}
			if test.fieldType.IsSet(opVal) {
				return types.True
			}
			return types.False
		}
	}

	obj := test.op.Eval(ctx)
	tester, ok := obj.(traits.FieldTester)
	if ok {
		return tester.IsSet(test.field)
	}
	container, ok := obj.(traits.Container)
	if ok {
		return container.Contains(test.field)
	}
	return types.ValOrErr(obj, "invalid type for field selection.")

}

type evalConst struct {
	id  int64
	val ref.Val
}

// ID implements the Interpretable interface method.
func (cons *evalConst) ID() int64 {
	return cons.id
}

// Eval implements the Interpretable interface method.
func (cons *evalConst) Eval(ctx Activation) ref.Val {
	return cons.val
}

func (cons *evalConst) Value() ref.Val {
	return cons.val
}

type evalOr struct {
	id  int64
	lhs Interpretable
	rhs Interpretable
}

// ID implements the Interpretable interface method.
func (or *evalOr) ID() int64 {
	return or.id
}

// Eval implements the Interpretable interface method.
func (or *evalOr) Eval(ctx Activation) ref.Val {
	// short-circuit lhs.
	lVal := or.lhs.Eval(ctx)
	lBool, lok := lVal.(types.Bool)
	if lok && lBool == types.True {
		return types.True
	}
	// short-circuit on rhs.
	rVal := or.rhs.Eval(ctx)
	rBool, rok := rVal.(types.Bool)
	if rok && rBool == types.True {
		return types.True
	}
	// return if both sides are bool false.
	if lok && rok {
		return types.False
	}
	// TODO: return both values as a set if both are unknown or error.
	// prefer left unknown to right unknown.
	if types.IsUnknown(lVal) {
		return lVal
	}
	if types.IsUnknown(rVal) {
		return rVal
	}
	// If the left-hand side is non-boolean return it as the error.
	if types.IsError(lVal) {
		return lVal
	}
	return types.ValOrErr(rVal, "no such overload")
}

type evalAnd struct {
	id  int64
	lhs Interpretable
	rhs Interpretable
}

// ID implements the Interpretable interface method.
func (and *evalAnd) ID() int64 {
	return and.id
}

// Eval implements the Interpretable interface method.
func (and *evalAnd) Eval(ctx Activation) ref.Val {
	// short-circuit lhs.
	lVal := and.lhs.Eval(ctx)
	lBool, lok := lVal.(types.Bool)
	if lok && lBool == types.False {
		return types.False
	}
	// short-circuit on rhs.
	rVal := and.rhs.Eval(ctx)
	rBool, rok := rVal.(types.Bool)
	if rok && rBool == types.False {
		return types.False
	}
	// return if both sides are bool true.
	if lok && rok {
		return types.True
	}
	// TODO: return both values as a set if both are unknown or error.
	// prefer left unknown to right unknown.
	if types.IsUnknown(lVal) {
		return lVal
	}
	if types.IsUnknown(rVal) {
		return rVal
	}
	// If the left-hand side is non-boolean return it as the error.
	if types.IsError(lVal) {
		return lVal
	}
	return types.ValOrErr(rVal, "no such overload")
}

type evalEq struct {
	id  int64
	lhs Interpretable
	rhs Interpretable
}

// ID implements the Interpretable interface method.
func (eq *evalEq) ID() int64 {
	return eq.id
}

// Eval implements the Interpretable interface method.
func (eq *evalEq) Eval(ctx Activation) ref.Val {
	lVal := eq.lhs.Eval(ctx)
	rVal := eq.rhs.Eval(ctx)
	return lVal.Equal(rVal)
}

type evalNe struct {
	id  int64
	lhs Interpretable
	rhs Interpretable
}

// ID implements the Interpretable interface method.
func (ne *evalNe) ID() int64 {
	return ne.id
}

// Eval implements the Interpretable interface method.
func (ne *evalNe) Eval(ctx Activation) ref.Val {
	lVal := ne.lhs.Eval(ctx)
	rVal := ne.rhs.Eval(ctx)
	eqVal := lVal.Equal(rVal)
	eqBool, ok := eqVal.(types.Bool)
	if !ok {
		return types.ValOrErr(eqVal, "no such overload: _!=_")
	}
	return !eqBool
}

type evalZeroArity struct {
	id   int64
	impl functions.FunctionOp
}

// ID implements the Interpretable interface method.
func (zero *evalZeroArity) ID() int64 {
	return zero.id
}

// Eval implements the Interpretable interface method.
func (zero *evalZeroArity) Eval(ctx Activation) ref.Val {
	return zero.impl()
}

type evalUnary struct {
	id       int64
	function string
	overload string
	arg      Interpretable
	trait    int
	impl     functions.UnaryOp
}

// ID implements the Interpretable interface method.
func (un *evalUnary) ID() int64 {
	return un.id
}

// Eval implements the Interpretable interface method.
func (un *evalUnary) Eval(ctx Activation) ref.Val {
	argVal := un.arg.Eval(ctx)
	// Early return if the argument to the function is unknown or error.
	if types.IsUnknownOrError(argVal) {
		return argVal
	}
	// If the implementation is bound and the argument value has the right traits required to
	// invoke it, then call the implementation.
	if un.impl != nil && (un.trait == 0 || argVal.Type().HasTrait(un.trait)) {
		return un.impl(argVal)
	}
	// Otherwise, if the argument is a ReceiverType attempt to invoke the receiver method on the
	// operand (arg0).
	if argVal.Type().HasTrait(traits.ReceiverType) {
		return argVal.(traits.Receiver).Receive(un.function, un.overload, []ref.Val{})
	}
	return types.NewErr("no such overload: %s", un.function)
}

type evalBinary struct {
	id       int64
	function string
	overload string
	lhs      Interpretable
	rhs      Interpretable
	trait    int
	impl     functions.BinaryOp
}

// ID implements the Interpretable interface method.
func (bin *evalBinary) ID() int64 {
	return bin.id
}

// Eval implements the Interpretable interface method.
func (bin *evalBinary) Eval(ctx Activation) ref.Val {
	lVal := bin.lhs.Eval(ctx)
	rVal := bin.rhs.Eval(ctx)
	// Early return if any argument to the function is unknown or error.
	if types.IsUnknownOrError(lVal) {
		return lVal
	}
	if types.IsUnknownOrError(rVal) {
		return rVal
	}
	// If the implementation is bound and the argument value has the right traits required to
	// invoke it, then call the implementation.
	if bin.impl != nil && (bin.trait == 0 || lVal.Type().HasTrait(bin.trait)) {
		return bin.impl(lVal, rVal)
	}
	// Otherwise, if the argument is a ReceiverType attempt to invoke the receiver method on the
	// operand (arg0).
	if lVal.Type().HasTrait(traits.ReceiverType) {
		return lVal.(traits.Receiver).Receive(bin.function, bin.overload, []ref.Val{rVal})
	}
	return types.NewErr("no such overload: %s", bin.function)
}

type evalVarArgs struct {
	id       int64
	function string
	overload string
	args     []Interpretable
	trait    int
	impl     functions.FunctionOp
}

// ID implements the Interpretable interface method.
func (fn *evalVarArgs) ID() int64 {
	return fn.id
}

// Eval implements the Interpretable interface method.
func (fn *evalVarArgs) Eval(ctx Activation) ref.Val {
	argVals := make([]ref.Val, len(fn.args), len(fn.args))
	// Early return if any argument to the function is unknown or error.
	for i, arg := range fn.args {
		argVals[i] = arg.Eval(ctx)
		if types.IsUnknownOrError(argVals[i]) {
			return argVals[i]
		}
	}
	// If the implementation is bound and the argument value has the right traits required to
	// invoke it, then call the implementation.
	arg0 := argVals[0]
	if fn.impl != nil && (fn.trait == 0 || arg0.Type().HasTrait(fn.trait)) {
		return fn.impl(argVals...)
	}
	// Otherwise, if the argument is a ReceiverType attempt to invoke the receiver method on the
	// operand (arg0).
	if arg0.Type().HasTrait(traits.ReceiverType) {
		return arg0.(traits.Receiver).Receive(fn.function, fn.overload, argVals[1:])
	}
	return types.NewErr("no such overload: %s", fn.function)
}

type evalList struct {
	id      int64
	elems   []Interpretable
	adapter ref.TypeAdapter
}

// ID implements the Interpretable interface method.
func (l *evalList) ID() int64 {
	return l.id
}

// Eval implements the Interpretable interface method.
func (l *evalList) Eval(ctx Activation) ref.Val {
	elemVals := make([]ref.Val, len(l.elems), len(l.elems))
	// If any argument is unknown or error early terminate.
	for i, elem := range l.elems {
		elemVal := elem.Eval(ctx)
		if types.IsUnknownOrError(elemVal) {
			return elemVal
		}
		elemVals[i] = elemVal
	}
	return types.NewDynamicList(l.adapter, elemVals)
}

type evalMap struct {
	id      int64
	keys    []Interpretable
	vals    []Interpretable
	adapter ref.TypeAdapter
}

// ID implements the Interpretable interface method.
func (m *evalMap) ID() int64 {
	return m.id
}

// Eval implements the Interpretable interface method.
func (m *evalMap) Eval(ctx Activation) ref.Val {
	entries := make(map[ref.Val]ref.Val)
	// If any argument is unknown or error early terminate.
	for i, key := range m.keys {
		keyVal := key.Eval(ctx)
		if types.IsUnknownOrError(keyVal) {
			return keyVal
		}
		valVal := m.vals[i].Eval(ctx)
		if types.IsUnknownOrError(valVal) {
			return valVal
		}
		entries[keyVal] = valVal
	}
	return types.NewDynamicMap(m.adapter, entries)
}

type evalObj struct {
	id       int64
	typeName string
	fields   []string
	vals     []Interpretable
	provider ref.TypeProvider
}

// ID implements the Interpretable interface method.
func (o *evalObj) ID() int64 {
	return o.id
}

// Eval implements the Interpretable interface method.
func (o *evalObj) Eval(ctx Activation) ref.Val {
	fieldVals := make(map[string]ref.Val)
	// If any argument is unknown or error early terminate.
	for i, field := range o.fields {
		val := o.vals[i].Eval(ctx)
		if types.IsUnknownOrError(val) {
			return val
		}
		fieldVals[field] = val
	}
	return o.provider.NewValue(o.typeName, fieldVals)
}

type evalFold struct {
	id        int64
	accuVar   string
	iterVar   string
	iterRange Interpretable
	accu      Interpretable
	cond      Interpretable
	step      Interpretable
	result    Interpretable
}

// ID implements the Interpretable interface method.
func (fold *evalFold) ID() int64 {
	return fold.id
}

// Eval implements the Interpretable interface method.
func (fold *evalFold) Eval(ctx Activation) ref.Val {
	foldRange := fold.iterRange.Eval(ctx)
	if !foldRange.Type().HasTrait(traits.IterableType) {
		return types.ValOrErr(foldRange, "got '%T', expected iterable type", foldRange)
	}
	// Configure the fold activation with the accumulator initial value.
	accuCtx := varActivationPool.Get().(*varActivation)
	accuCtx.parent = ctx
	accuCtx.name = fold.accuVar
	accuCtx.val = fold.accu.Eval(ctx)
	iterCtx := varActivationPool.Get().(*varActivation)
	iterCtx.parent = accuCtx
	iterCtx.name = fold.iterVar
	it := foldRange.(traits.Iterable).Iterator()
	for it.HasNext() == types.True {
		// Modify the iter var in the fold activation.
		iterCtx.val = it.Next()

		// Evaluate the condition, terminate the loop if false.
		cond := fold.cond.Eval(iterCtx)
		condBool, ok := cond.(types.Bool)
		if !types.IsUnknown(cond) && ok && condBool != types.True {
			break
		}

		// Evalute the evaluation step into accu var.
		accuCtx.val = fold.step.Eval(iterCtx)
	}
	// Compute the result.
	res := fold.result.Eval(accuCtx)
	varActivationPool.Put(iterCtx)
	varActivationPool.Put(accuCtx)
	return res
}

// Optional Intepretable implementations that specialize, subsume, or extend the core evaluation
// plan via decorators.

// evalSetMembership is an Interpretable implementation which tests whether an input value
// exists within the set of map keys used to model a set.
type evalSetMembership struct {
	inst        Interpretable
	arg         Interpretable
	argTypeName string
	valueSet    map[ref.Val]ref.Val
}

// ID implements the Interpretable interface method.
func (e *evalSetMembership) ID() int64 {
	return e.inst.ID()
}

// Eval implements the Interpretable interface method.
func (e *evalSetMembership) Eval(ctx Activation) ref.Val {
	val := e.arg.Eval(ctx)
	if val.Type().TypeName() != e.argTypeName {
		return types.ValOrErr(val, "no such overload")
	}
	if ret, found := e.valueSet[val]; found {
		return ret
	}
	return types.False
}

// evalWatch is an Interpretable implementation that wraps the execution of a given
// expression so that it may observe the computed value and send it to an observer.
type evalWatch struct {
	inst     Interpretable
	observer evalObserver
}

// ID implements the Interpretable interface method.
func (e *evalWatch) ID() int64 {
	return e.inst.ID()
}

// Eval implements the Interpretable interface method.
func (e *evalWatch) Eval(ctx Activation) ref.Val {
	val := e.inst.Eval(ctx)
	e.observer(e.inst.ID(), val)
	return val
}

// evalWatchAttr describes a watcher of an instAttr Interpretable.
//
// Since the watcher may be selected against at a later stage in program planning, the watcher
// must implement the instAttr interface by proxy.
type evalWatchAttr struct {
	instAttr
	observer evalObserver
}

// Eval implements the Interpretable interface method.
func (e *evalWatchAttr) Eval(vars Activation) ref.Val {
	val := e.instAttr.Eval(vars)
	e.observer(e.instAttr.ID(), val)
	return val
}

// evalWatchConst describes a watcher of an instConst Interpretable.
type evalWatchConst struct {
	instConst
	observer evalObserver
}

// Eval implements the Interpretable interface method.
func (e *evalWatchConst) Eval(vars Activation) ref.Val {
	val := e.instConst.Value()
	e.observer(e.instConst.ID(), val)
	return val
}

// evalExhaustiveOr is just like evalOr, but does not short-circuit argument evaluation.
type evalExhaustiveOr struct {
	id  int64
	lhs Interpretable
	rhs Interpretable
}

// ID implements the Interpretable interface method.
func (or *evalExhaustiveOr) ID() int64 {
	return or.id
}

// Eval implements the Interpretable interface method.
func (or *evalExhaustiveOr) Eval(ctx Activation) ref.Val {
	lVal := or.lhs.Eval(ctx)
	rVal := or.rhs.Eval(ctx)
	lBool, lok := lVal.(types.Bool)
	if lok && lBool == types.True {
		return types.True
	}
	rBool, rok := rVal.(types.Bool)
	if rok && rBool == types.True {
		return types.True
	}
	if lok && rok {
		return types.False
	}
	if types.IsUnknown(lVal) {
		return lVal
	}
	if types.IsUnknown(rVal) {
		return rVal
	}
	// TODO: Combine the errors into a set in the future.
	// If the left-hand side is non-boolean return it as the error.
	if types.IsError(lVal) {
		return lVal
	}
	return types.ValOrErr(rVal, "no such overload")
}

// evalExhaustiveAnd is just like evalAnd, but does not short-circuit argument evaluation.
type evalExhaustiveAnd struct {
	id  int64
	lhs Interpretable
	rhs Interpretable
}

// ID implements the Interpretable interface method.
func (and *evalExhaustiveAnd) ID() int64 {
	return and.id
}

// Eval implements the Interpretable interface method.
func (and *evalExhaustiveAnd) Eval(ctx Activation) ref.Val {
	lVal := and.lhs.Eval(ctx)
	rVal := and.rhs.Eval(ctx)
	lBool, lok := lVal.(types.Bool)
	if lok && lBool == types.False {
		return types.False
	}
	rBool, rok := rVal.(types.Bool)
	if rok && rBool == types.False {
		return types.False
	}
	if lok && rok {
		return types.True
	}
	if types.IsUnknown(lVal) {
		return lVal
	}
	if types.IsUnknown(rVal) {
		return rVal
	}
	// TODO: Combine the errors into a set in the future.
	// If the left-hand side is non-boolean return it as the error.
	if types.IsError(lVal) {
		return lVal
	}
	return types.ValOrErr(rVal, "no such overload")
}

// evalExhaustiveConditional is like evalConditional, but does not short-circuit argument
// evaluation.
type evalExhaustiveConditional struct {
	id      int64
	adapter ref.TypeAdapter
	attr    *conditionalAttribute
}

// ID implements the Interpretable interface method.
func (cond *evalExhaustiveConditional) ID() int64 {
	return cond.id
}

// Eval implements the Interpretable interface method.
func (cond *evalExhaustiveConditional) Eval(ctx Activation) ref.Val {
	cVal := cond.attr.expr.Eval(ctx)
	tVal, err := cond.attr.truthy.Resolve(ctx)
	if err != nil {
		return types.NewErr(err.Error())
	}
	fVal, err := cond.attr.falsy.Resolve(ctx)
	if err != nil {
		return types.NewErr(err.Error())
	}
	cBool, ok := cVal.(types.Bool)
	if !ok {
		return types.ValOrErr(cVal, "no such overload")
	}
	if cBool {
		return cond.adapter.NativeToValue(tVal)
	}
	return cond.adapter.NativeToValue(fVal)
}

// evalExhaustiveFold is like evalFold, but does not short-circuit argument evaluation.
type evalExhaustiveFold struct {
	id        int64
	accuVar   string
	iterVar   string
	iterRange Interpretable
	accu      Interpretable
	cond      Interpretable
	step      Interpretable
	result    Interpretable
}

// ID implements the Interpretable interface method.
func (fold *evalExhaustiveFold) ID() int64 {
	return fold.id
}

// Eval implements the Interpretable interface method.
func (fold *evalExhaustiveFold) Eval(ctx Activation) ref.Val {
	foldRange := fold.iterRange.Eval(ctx)
	if !foldRange.Type().HasTrait(traits.IterableType) {
		return types.ValOrErr(foldRange, "got '%T', expected iterable type", foldRange)
	}
	// Configure the fold activation with the accumulator initial value.
	accuCtx := varActivationPool.Get().(*varActivation)
	accuCtx.parent = ctx
	accuCtx.name = fold.accuVar
	accuCtx.val = fold.accu.Eval(ctx)
	iterCtx := varActivationPool.Get().(*varActivation)
	iterCtx.parent = accuCtx
	iterCtx.name = fold.iterVar
	it := foldRange.(traits.Iterable).Iterator()
	for it.HasNext() == types.True {
		// Modify the iter var in the fold activation.
		iterCtx.val = it.Next()

		// Evaluate the condition, but don't terminate the loop as this is exhaustive eval!
		fold.cond.Eval(iterCtx)

		// Evalute the evaluation step into accu var.
		accuCtx.val = fold.step.Eval(iterCtx)
	}
	// Compute the result.
	res := fold.result.Eval(accuCtx)
	varActivationPool.Put(iterCtx)
	varActivationPool.Put(accuCtx)
	return res
}

// instConst interface for tracking whether the Interpretable is a constant value.
type instConst interface {
	Interpretable

	// Value returns the constant value of the instruction.
	Value() ref.Val
}

// instAttr interface for tracking whether the Interpretable is an attribute.
type instAttr interface {
	Interpretable

	// Attr returns the Attribute value.
	Attr() Attribute

	// Adapter returns the type adapter to be used for adapting resolved Attribute values.
	Adapter() ref.TypeAdapter

	// AddQualifier proxies the Attribute.AddQualifier method.
	AddQualifier(Qualifier) (instAttr, error)
}

type evalAttr struct {
	adapter ref.TypeAdapter
	attr    Attribute
}

// ID of the attribute instruction.
func (a *evalAttr) ID() int64 {
	return a.attr.ID()
}

// Attr implements the instAttr interface method.
func (a *evalAttr) Attr() Attribute {
	return a.attr
}

// Adapter implements the instAttr interface method.
func (a *evalAttr) Adapter() ref.TypeAdapter {
	return a.adapter
}

// Eval implements the Interpretable interface method.
func (a *evalAttr) Eval(ctx Activation) ref.Val {
	v, err := a.attr.Resolve(ctx)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return a.adapter.NativeToValue(v)
}

// AddQualifier implements the instAttr interface method.
func (a *evalAttr) AddQualifier(qual Qualifier) (instAttr, error) {
	_, err := a.attr.AddQualifier(qual)
	return a, err
}
