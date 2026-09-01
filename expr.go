// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/vm"
	"github.com/kelindar/expr/assignment"
	"github.com/kelindar/expr/collection"
	"github.com/kelindar/expr/encoding"
	"github.com/kelindar/expr/network"
	"github.com/kelindar/expr/numeric"
	"github.com/kelindar/expr/text"
	"github.com/kelindar/expr/validate"
	"github.com/tidwall/gjson"
)

const (
	maxSourceBytes = 8 << 10
	maxASTNodes    = 1000
	maxJSONBytes   = 256 << 10
	maxEntries     = 10000
	maxNesting     = 64
	maxVMMemory    = 1_000_000
)

var compileOptions = options()

var evaluatorPool = sync.Pool{New: func() any {
	return &vm.VM{MemoryBudget: maxVMMemory}
}}

// Type is a statically known expression result type.
type Type string

// Program is a compiled expression program.
type Program struct {
	source string
	prog   *vm.Program
	json   jsonProjection
	bool   *boolPredicate
	uses   envUsage
}

// Type returns the statically known JSON Schema type of the result.
// An empty string means the expression result is dynamic.
func (p *Program) Type() Type {
	if p == nil || p.prog == nil {
		return ""
	}
	typ := p.prog.Node().Type()
	if typ == nil {
		return ""
	}
	switch typ.Kind() {
	case reflect.Bool:
		return Type("boolean")
	case reflect.String:
		return Type("string")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return Type("integer")
	case reflect.Float32, reflect.Float64:
		return Type("number")
	case reflect.Array, reflect.Slice:
		return Type("array")
	case reflect.Map:
		return Type("object")
	default:
		return ""
	}
}

// Compile compiles an expression program by source.
func Compile(source string) (*Program, error) {
	switch {
	case strings.TrimSpace(source) == "":
		return nil, fmt.Errorf("expression: source is required")
	case len(source) > maxSourceBytes:
		return nil, fmt.Errorf("expression: source exceeds %d bytes", maxSourceBytes)
	}

	prog, err := expr.Compile(source, compileOptions...)
	if err != nil {
		return nil, fmt.Errorf("expression: compile: %w", err)
	}

	return &Program{source: source, prog: prog, json: compileJSON(source), bool: compileBool(source), uses: inspectEnv(prog)}, nil
}

func options() []expr.Option {
	options := []expr.Option{
		expr.Env(env{}),
		expr.AllowUndefinedVariables(),
		expr.MaxNodes(maxASTNodes),
		expr.Patch(patcher{}),
	}
	options = append(options, numeric.Options()...)
	options = append(options, collection.Options()...)
	options = append(options, text.Options()...)
	options = append(options, encoding.Options()...)
	options = append(options, network.Options()...)
	options = append(options, validate.Options()...)
	options = append(options, assignment.Options()...)
	return options
}

// JSON evaluates the compiled expression into JSON.
func (p *Program) JSON(input []byte) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("expression: program is required")
	}
	input, err := validateInput(input)
	if err != nil {
		return nil, err
	}
	if p.json != nil {
		out, err := p.json.evalValid(input)
		if err != errJSONFallback {
			if err == nil {
				err = validateOutput(out)
			}
			return out, err
		}
	}
	value, err := p.Eval(nil, input, time.Time{})
	if err != nil {
		return nil, err
	}
	// Eval normalizes values before this point; outputJSONValue supplies the
	// stable JSON representation for time and duration values.
	out, err := json.Marshal(outputJSONValue(value))
	if err != nil {
		return nil, fmt.Errorf("expression: result: %w", err)
	}
	if err := validateOutput(out); err != nil {
		return nil, err
	}
	return out, nil
}

// AppendJSON evaluates the compiled expression and appends its JSON result to dst.
func (p *Program) AppendJSON(dst, input []byte) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("expression: program is required")
	}
	input, err := validateInput(input)
	if err != nil {
		return nil, err
	}
	if p.json != nil {
		start := len(dst)
		out, err := p.json.appendValid(dst, input)
		if err != errJSONFallback {
			if err != nil {
				return dst[:start], err
			}
			if err := validateOutput(out[start:]); err != nil {
				return dst[:start], err
			}
			return out, nil
		}
	}
	out, err := p.JSON(input)
	return append(dst, out...), err
}

// Eval evaluates the compiled expression against JSON input and context at now.
// A zero now captures the current UTC time. The context is optional JSON metadata.
func (p *Program) Eval(ctx, input jsontext.Value, now time.Time) (any, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	return evalAt(p, input, ctx, now)
}

func evalAt(program *Program, input, context []byte, captured time.Time) (any, error) {
	if program == nil {
		return nil, fmt.Errorf("expression: program is required")
	}
	var err error
	input, err = validateInput(input)
	if err != nil {
		return nil, err
	}
	context, err = validateInput(context)
	if err != nil {
		return nil, fmt.Errorf("expression: context: %w", err)
	}
	root := value{raw: input}
	ctx := value{raw: context}
	runtimeEnv := env{
		This:     root,
		Context:  ctx,
		Raw:      fnRaw,
		Time:     fnTime,
		Duration: fnDuration,
	}
	if program.uses&usesInput != 0 {
		runtimeEnv.Get = getFunc(root)
	}
	if program.uses&usesContext != 0 {
		runtimeEnv.ContextGet = getFunc(ctx)
	}
	if program.uses&usesJSON != 0 {
		runtimeEnv.JSON = jsonFunc(root)
	}
	if program.uses&usesNow != 0 {
		runtimeEnv.Now = func() any { return captured }
	}
	runner := evaluatorPool.Get().(*vm.VM)
	out, err := runner.Run(program.prog, runtimeEnv)
	releaseEvaluator(runner)
	if err != nil {
		return nil, fmt.Errorf("expression: eval: %w", err)
	}
	out = unwrap(out)
	if out, err = normalizeValue(out, 1, new(int)); err != nil {
		return nil, err
	}
	if err := encodedResult(out); err != nil {
		return nil, err
	}
	return out, nil
}

type envUsage uint8

const (
	usesInput envUsage = 1 << iota
	usesContext
	usesJSON
	usesNow
)

func inspectEnv(program *vm.Program) envUsage {
	var uses envUsage
	node := program.Node()
	ast.Walk(&node, &uses)
	return uses
}

func (uses *envUsage) Visit(node *ast.Node) {
	identifier, ok := (*node).(*ast.IdentifierNode)
	if !ok {
		return
	}
	switch identifier.Value {
	case "__get":
		*uses |= usesInput
	case "__context_get":
		*uses |= usesContext
	case "json":
		*uses |= usesJSON
	case "now":
		*uses |= usesNow
	}
}

func releaseEvaluator(runner *vm.VM) {
	clear(runner.Stack[:cap(runner.Stack)])
	clear(runner.Scopes[:cap(runner.Scopes)])
	clear(runner.Variables)
	runner.MemoryBudget = maxVMMemory
	evaluatorPool.Put(runner)
}

// Bool evaluates the compiled expression and requires a boolean result.
func (p *Program) Bool(input []byte) (bool, error) {
	input, err := validateInput(input)
	if err != nil {
		return false, err
	}
	if p != nil && p.bool != nil && gjson.ValidBytes(input) {
		if value, ok := p.bool.eval(input); ok {
			return value, nil
		}
	}
	out, err := p.Eval(nil, input, time.Time{})
	if err != nil {
		return false, err
	}
	v, ok := out.(bool)
	if !ok {
		return false, fmt.Errorf("expression: result must be bool, got %T", out)
	}
	return v, nil
}

type env struct {
	This       any                            `expr:"this"`
	Context    any                            `expr:"context"`
	Get        func(string) any               `expr:"__get"`
	ContextGet func(string) any               `expr:"__context_get"`
	JSON       func(args ...any) (any, error) `expr:"json"`
	Raw        func(any) (string, error)      `expr:"raw"`
	Time       func(any) (any, error)         `expr:"time"`
	Now        func() any                     `expr:"now"`
	Duration   func(string) (any, error)      `expr:"duration"`
}

func location(err error) (line, column int, ok bool) {
	var fileErr *file.Error
	if !errors.As(err, &fileErr) {
		return 0, 0, false
	}
	return fileErr.Line, fileErr.Column + 1, fileErr.Line > 0
}
