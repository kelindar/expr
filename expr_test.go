// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type fixture struct {
	Name   string         `yaml:"name"`
	Expr   string         `yaml:"expr"`
	Input  map[string]any `yaml:"input"`
	Expect any            `yaml:"expect"`
	Error  bool           `yaml:"error"`
}

func TestCompile(t *testing.T) {
	t.Run("compiles independently", func(t *testing.T) {
		a, err := Compile(`this.state == "done"`)
		require.NoError(t, err)
		b, err := Compile(`this.state == "done"`)
		require.NoError(t, err)
		require.NotSame(t, a, b)
	})

	t.Run("rejects empty", func(t *testing.T) {
		_, err := Compile("")
		require.Error(t, err)
	})

	t.Run("reports static JSON result types", func(t *testing.T) {
		for _, tc := range []struct {
			source string
			want   Type
		}{
			{source: `is(this, "email")`, want: "boolean"},
			{source: `42`, want: "integer"},
			{source: `1.5`, want: "number"},
			{source: `"hello"`, want: "string"},
			{source: `[1, 2]`, want: "array"},
			{source: `{ok: true}`, want: "object"},
			{source: `this.value`, want: ""},
		} {
			t.Run(tc.source, func(t *testing.T) {
				program, err := Compile(tc.source)
				require.NoError(t, err)
				assert.Equal(t, tc.want, program.Type())
			})
		}
		assert.Empty(t, (*Program)(nil).Type())
	})
}

func TestFixtures(t *testing.T) {
	for _, tc := range readYAMLFile[fixture](t, filepath.Join("internal", "fixtures", "expr", "basic.yaml")) {
		t.Run(tc.Name, func(t *testing.T) {
			program, err := Compile(tc.Expr)
			require.NoError(t, err)
			input, err := json.Marshal(tc.Input)
			require.NoError(t, err)

			got, err := program.Bool(input)
			if tc.Error {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.Expect, got)
		})
	}
}

func readYAMLFile[T any](t *testing.T, file string) []T {
	t.Helper()

	body, err := os.ReadFile(file)
	require.NoError(t, err)

	var out []T
	require.NoError(t, yaml.Unmarshal(body, &out))
	return out
}

func TestCompileJSON(t *testing.T) {
	_, err := Compile(strings.Repeat("1+", maxSourceBytes/2) + "1")
	require.Error(t, err)
	_, err = Compile("this.")
	require.Error(t, err)
	_, err = Compile("   ")
	require.Error(t, err)
	_, err = (*Program)(nil).JSON(nil)
	require.Error(t, err)
	_, err = (*Program)(nil).AppendJSON(nil, nil)
	require.Error(t, err)
	program, err := Compile(`{copy: this, name: trim(this.name), missing: this.missing, literal: nil}`)
	require.NoError(t, err)
	out, err := program.JSON([]byte(`{"name":" Ada "}`))
	require.NoError(t, err)
	require.JSONEq(t, `{"copy":{"name":" Ada "},"name":"Ada","missing":null,"literal":null}`, string(out))
	appended, err := program.AppendJSON([]byte("prefix"), []byte(`{"name":" Ada "}`))
	require.NoError(t, err)
	require.Equal(t, "prefix"+string(out), string(appended))
	program, err = Compile(`{name: trim(this.name)}`)
	require.NoError(t, err)
	_, err = program.JSON([]byte(`{}`))
	require.Error(t, err)
	program, err = Compile(`{value: trim(this.value)}`)
	require.NoError(t, err)
	_, err = program.JSON([]byte(`{"value":42}`))
	require.Error(t, err)
	projection := jsonProjection{{key: `"x"`, literal: []byte(`1`)}}
	_, err = projection.evalValid([]byte(`{"a":1}`))
	require.NoError(t, err)
}

func TestLimitsAndNormalization(t *testing.T) {
	got, err := validateInput(nil)
	require.NoError(t, err)
	require.Equal(t, []byte("{}"), got)
	_, err = validateInput([]byte(strings.Repeat("x", maxJSONBytes+1)))
	require.Error(t, err)
	for _, raw := range [][]byte{[]byte(`{`), []byte(`1 2`)} {
		_, err := validateInput(raw)
		require.Error(t, err)
	}
	err = validateOutput([]byte(strings.Repeat("x", maxJSONBytes+1)))
	require.Error(t, err)
	err = validateOutput([]byte(`{`))
	require.Error(t, err)
	require.NoError(t, Validate([]byte(`{"ok":true}`)))
	for _, value := range []any{nil, true, "x", time.Unix(0, 0), time.Duration(time.Second), jsontext.Value("42"), jsontext.Value("1.5"), int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1), []any{1}, map[string]any{"x": 1}, [1]int{1}, [1]string{"x"}, map[string]int{"x": 1}} {
		_, err := normalizeValue(value, 1, new(int))
		require.NoError(t, err)
	}
	for _, value := range []any{jsontext.Value("bad"), jsontext.Value("NaN"), math.NaN(), math.Inf(1), []byte("x"), map[int]int{1: 2}, func() {}} {
		_, err := normalizeValue(value, 1, new(int))
		require.Error(t, err)
	}
	deep := any("leaf")
	for i := 0; i < maxNesting; i++ {
		deep = []any{deep}
	}
	err = validateValue(deep)
	require.Error(t, err)
	wide := make([]any, maxEntries+1)
	_, err = normalizeValue(wide, 1, new(int))
	require.Error(t, err)
	_, err = normalizeValue([]any{1}, maxNesting+1, new(int))
	require.Error(t, err)
}

func TestOutputLimitAtomic(t *testing.T) {
	input, err := json.Marshal(map[string]string{"value": strings.Repeat("x", 131100)})
	require.NoError(t, err)
	result := Evaluate(Request{Expression: "repeat(this.value, 2)", Input: input})
	require.Equal(t, "error", result.Type)
	require.Equal(t, "limit_exceeded", result.Out.(Failure).Code)

	projectionInput, err := json.Marshal(map[string]string{"value": strings.Repeat("x", 262130)})
	require.NoError(t, err)
	program, err := Compile(`{copy: this}`)
	require.NoError(t, err)
	_, err = program.JSON(projectionInput)
	require.Error(t, err)
	appended, err := program.AppendJSON([]byte("prefix"), projectionInput)
	require.Error(t, err)
	require.Equal(t, "prefix", string(appended))
}

func TestContractErrors(t *testing.T) {
	for _, tc := range []struct {
		expression string
		kind       string
	}{
		{"nil", "null"}, {"true", "boolean"}, {"42", "integer"}, {"1.5", "number"}, {`"x"`, "string"}, {"[1]", "array"}, {`{"x":1}`, "object"}, {"time(0)", "time"}, {`duration("1s")`, "duration"},
	} {
		result := Evaluate(Request{Expression: tc.expression, At: "2026-08-30T12:00:00+04:00"})
		require.Equal(t, tc.kind, result.Type, tc.expression)
	}
	for _, tc := range []Request{{}, {Expression: "1", Input: []byte(`{`)}, {Expression: "1", At: "bad"}, {Expression: "1 +"}, {Expression: "1/0"}} {
		result := Evaluate(tc)
		require.Equal(t, "error", result.Type)
		require.NotEmpty(t, result.Out)
	}
	for _, value := range []any{nil, true, int64(1), float64(1), "x", time.Unix(0, 0), time.Second, []any{1}, map[string]any{"x": 1}} {
		kind, _, err := result(value)
		require.NoError(t, err)
		require.NotEmpty(t, kind)
	}
	for _, value := range []any{math.NaN(), math.Inf(1), func() {}} {
		_, _, err := result(value)
		require.Error(t, err)
	}
	_, _, err := result(float32(math.NaN()))
	require.Error(t, err)
	_, _, err = result(strings.Repeat("x", maxJSONBytes))
	require.Error(t, err)
	_, _, err = result(struct{}{})
	require.Error(t, err)
	require.Equal(t, "limit_exceeded", inputCode(errors.New("expression: input exceeds limit")))
	require.Equal(t, "invalid_input", inputCode(errors.New("bad input")))
	require.Equal(t, "limit_exceeded", compileCode(errors.New("expression: source exceeds limit")))
	require.Equal(t, "compile_error", compileCode(errors.New("syntax")))
	require.Equal(t, "limit_exceeded", evalCode(errors.New("memory budget exceeded")))
	require.Equal(t, "limit_exceeded", evalCode(errors.New("expression: output exceeds 1")))
	require.Equal(t, "evaluation_error", evalCode(errors.New("expression: eval: bad")))
	line, column, ok := location(&file.Error{Line: 2, Column: 3})
	require.True(t, ok)
	require.Equal(t, 2, line)
	require.Equal(t, 4, column)
	_, _, ok = location(errors.New("plain"))
	require.False(t, ok)
}

func TestCatalogAndValue(t *testing.T) {
	functions := functions()
	require.NotEmpty(t, functions)
	for i := 1; i < len(functions); i++ {
		require.LessOrEqual(t, functions[i-1].Domain+functions[i-1].Name, functions[i].Domain+functions[i].Name)
	}
	for _, function := range functions {
		require.NotEmpty(t, function.Description, function.Name)
		require.NotEmpty(t, function.Usage, function.Name)
		require.NotEmpty(t, function.Returns, function.Name)
		require.NotEmpty(t, function.Example, function.Name)
	}
	reference := GetReference()
	require.Equal(t, "1.17.8", reference.Version)
	require.Contains(t, Guide(), "# Expression reference")
	value := value{raw: []byte(`{"x":1}`)}
	raw, err := value.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, `{"x":1}`, string(raw))
	value.path = "x"
	raw, err = value.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "1", string(raw))
	value.path = "missing"
	raw, err = value.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "null", string(raw))
}

func TestBoolInternals(t *testing.T) {
	for _, source := range []string{`this.x == 1`, `this.x != 1`, `this.x < 2`, `this.x <= 2`, `this.x > 0`, `this.x >= 0`, `1 <= this.x`, `this.x == "a"`, `this.x == true`, `this.x == nil`} {
		predicate := compileBool(source)
		require.NotNil(t, predicate, source)
		_, _ = predicate.eval([]byte(`{"x":1}`))
	}
	for _, source := range []string{`this.x + 1`, `this.a.b == 1`, `context.x == 1`, `this.x == [1]`} {
		require.Nil(t, compileBool(source))
	}
	for _, raw := range [][]byte{[]byte(`{"x":"a"}`), []byte(`{"x":true}`), []byte(`{"x":{}}`), []byte(`{"x":1.5}`), []byte(`{"x":999999999999999999999}`)} {
		predicate := compileBool(`this.x == 1`)
		_, _ = predicate.eval(raw)
	}
	for _, op := range []string{"==", "!=", "<", "<=", ">", ">="} {
		require.True(t, comparison(op))
		require.Equal(t, op, reverseComparison(reverseComparison(op)))
		_ = compareOrder(0, op)
	}
	require.False(t, comparison("bad"))
	require.Equal(t, "bad", reverseComparison("bad"))
	require.True(t, compareOrder(0, "bad"))
	require.True(t, (&boolPredicate{kind: 's'}).supports("<"))
	require.False(t, (&boolPredicate{kind: 'n'}).supports("<"))
	for _, node := range []ast.Node{&ast.NilNode{}, &ast.IntegerNode{Value: 1}, &ast.FloatNode{Value: 1}, &ast.StringNode{Value: "x"}, &ast.BoolNode{Value: true}} {
		_, ok := predicateLiteral(node)
		require.True(t, ok)
	}
}

func TestJSONProjectionBranches(t *testing.T) {
	for _, tc := range []struct {
		name string
		node ast.Node
		ok   bool
	}{
		{"this", &ast.IdentifierNode{Value: "this"}, true},
		{"member", &ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.StringNode{Value: "name"}}, true},
		{"literal", &ast.StringNode{Value: "x"}, true},
		{"nil", &ast.NilNode{}, true},
		{"integer", &ast.IntegerNode{Value: 1}, true},
		{"float", &ast.FloatNode{Value: 1.5}, true},
		{"bool", &ast.BoolNode{Value: true}, true},
		{"trim", &ast.BuiltinNode{Name: "trim", Arguments: []ast.Node{&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.StringNode{Value: "name"}}}}, true},
	} {
		field, ok := compileJSONField(tc.name, tc.node)
		require.Equal(t, tc.ok, ok, tc.name)
		if ok {
			require.NotEmpty(t, field.key)
		}
	}
	_, ok := compileJSONField("trim", &ast.BuiltinNode{Name: "trim", Arguments: []ast.Node{&ast.StringNode{Value: "x"}}})
	require.False(t, ok)
	_, ok = compileJSONField("call", &ast.CallNode{})
	require.False(t, ok)
	_, ok = compileJSONField("badtrim", &ast.BuiltinNode{Name: "trim", Arguments: nil})
	require.False(t, ok)
	_, ok = compileJSONField("unsupported", &ast.IdentifierNode{Value: "other"})
	require.False(t, ok)
	_, ok = compileJSONField("key", &ast.StringNode{Value: "x"})
	require.True(t, ok)
	require.Nil(t, compileJSON(`{"x": 1, "x": 2}`))
	require.Nil(t, compileJSON(`{other: foo()}`))
	require.Nil(t, compileJSON("[1]"))
	require.Nil(t, compileJSON("not valid"))
	_ = compileJSON(`{1: 2}`)

	for _, node := range []ast.Node{
		&ast.IdentifierNode{Value: "this"},
		&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.StringNode{Value: "a"}},
		&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.IntegerNode{Value: 0}},
		&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.StringNode{Value: "a.b"}},
		&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.StringNode{Value: "a"}, Optional: true},
		&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.StringNode{Value: "a"}, Method: true},
	} {
		_ = jsonKeys(node)
		_, _ = jsonPath(node)
	}
	_ = jsonKeys(&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.BoolNode{Value: true}})
	_, _ = jsonPath(&ast.StringNode{Value: "x"})

	field := jsonField{path: "name"}
	require.Equal(t, []byte(`"Ada"`), field.appendValue(nil, []byte(`{"name":"Ada"}`)))
	require.Equal(t, []byte(`null`), field.appendValue(nil, []byte(`{}`)))
	field.trim = true
	require.Equal(t, []byte(`"Ada"`), field.appendValue(nil, []byte(`{"name":" Ada "}`)))
	require.Nil(t, field.appendValue(nil, []byte(`{"name":42}`)))
	require.Nil(t, field.appendValue(nil, []byte(`{}`)))
	field.trim = false
	require.Equal(t, []byte(`"Ada"`), field.appendValue(nil, []byte(`{"name":"Ada"}`)))
	field.path = ""
	require.Equal(t, []byte(`null`), field.appendValue(nil, nil))
	field.trim = true
	require.Nil(t, field.appendValue(nil, []byte(`42`)))
	field.trim = false
	require.Equal(t, []byte(`42`), field.appendValue(nil, []byte(`42`)))
	field.path = "missing"
	require.Equal(t, []byte(`null`), field.appendValue(nil, []byte(`{"x":1}`)))
	field.trim = true
	require.Nil(t, field.appendValue(nil, []byte(`{"x":1}`)))
	nonTrimPath := jsonField{path: "n"}
	require.Equal(t, []byte(`42`), nonTrimPath.appendValue(nil, []byte(`{"n":42}`)))
	field = jsonField{path: "name", trim: true}
	require.NotNil(t, field.appendValue(nil, []byte(`{"name":" \\u0041da "}`)))
	_ = field.appendValue(nil, []byte(`{"name":"\\uZZZZ"}`))
	rootTrim := jsonField{trim: true}
	require.Equal(t, []byte(`"Ada"`), rootTrim.appendValue(nil, []byte(`" Ada "`)))
	projection := jsonProjection{{key: `"x"`, path: "name"}}
	require.NoError(t, func() error { _, err := projection.evalValid([]byte(`{"name":"x"}`)); return err }())
}

func TestBuiltinContext(t *testing.T) {
	root := value{raw: []byte(`{"x":1,"str":"  x "}`)}
	jsonFn := jsonFunc(root)
	gotJSON, err := jsonFn(`{"x":1}`, "x")
	require.NoError(t, err)
	require.Equal(t, int64(1), gotJSON)
	_, err = jsonFn(value{raw: []byte(`{"x":1}`), path: "x"}, "x")
	require.Error(t, err)
	_, err = fnRaw(1)
	require.Error(t, err)
	_, err = jsonRaw("{" + "x")
	require.NoError(t, err)
	_, err = jsonRaw(value{raw: root.raw, path: "x"})
	require.Error(t, err)
	_, err = jsonRaw(value{raw: root.raw, path: "str"})
	require.NoError(t, err)
	_, err = jsonRaw(value{raw: root.raw, path: "missing"})
	require.Error(t, err)
	_, err = jsonRaw(value{raw: root.raw})
	require.NoError(t, err)
	_, err = jsonFn(1)
	require.Error(t, err)
	program, err := Compile(`this.x`)
	require.NoError(t, err)
	_, err = program.Eval([]byte(`{`), []byte(`{"x":1}`), time.Time{})
	require.Error(t, err)
	_, err = (*Program)(nil).Eval(nil, nil, time.Time{})
	require.Error(t, err)
	resultRaw, err := program.JSON([]byte(`{"x":1}`))
	require.NoError(t, err)
	require.JSONEq(t, `1`, string(resultRaw))
	appended, err := program.AppendJSON([]byte("p"), []byte(`{"x":1}`))
	require.NoError(t, err)
	require.Equal(t, `p1`, string(appended))
	empty := value{}
	raw, err := empty.MarshalJSON()
	require.NoError(t, err)
	require.Equal(t, "null", string(raw))
	require.Nil(t, decodeJSON("{"))
}

func TestRemainingInternalBranches(t *testing.T) {
	for _, value := range []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1)} {
		kind, _, err := result(value)
		require.NoError(t, err)
		require.Equal(t, "integer", kind)
	}
	_, _, err := result(float32(1.5))
	require.NoError(t, err)
	require.Error(t, encodedResult(map[string]any{"bad": func() {}}))
	_, err = normalizeValue(map[string]any{"bad": func() {}}, 1, new(int))
	require.Error(t, err)
	_, err = normalizeValue([]any{func() {}}, 1, new(int))
	require.Error(t, err)
	require.Error(t, validateJSON(nil))
	require.Error(t, validateJSON([]byte("1e999")))
	nestedJSON := strings.Repeat("[", maxNesting+1) + "0" + strings.Repeat("]", maxNesting+1)
	require.Error(t, validateJSON([]byte(nestedJSON)))
	_, err = normalizeValue([maxEntries + 1]int{}, 1, new(int))
	require.Error(t, err)
	_, err = normalizeValue([1]func(){}, 1, new(int))
	require.Error(t, err)
	wideMap := make(map[string]any, maxEntries+1)
	for i := 0; i <= maxEntries; i++ {
		wideMap[strconv.Itoa(i)] = i
	}
	_, err = normalizeValue(wideMap, 1, new(int))
	require.Error(t, err)
	for _, source := range []string{`this.x > 1`, `this.x == "x"`, `this.x == true`, `this.x == nil`} {
		predicate := compileBool(source)
		for _, raw := range [][]byte{[]byte(`{}`), []byte(`{"x":null}`), []byte(`{"x":1.5}`), []byte(`{"x":"x"}`), []byte(`{"x":true}`)} {
			_, _ = predicate.eval(raw)
		}
	}
	rootPredicate := &boolPredicate{kind: 'i', integer: 1, op: "=="}
	_, ok := rootPredicate.eval([]byte(`1`))
	require.True(t, ok)
	_, ok = pathOf(&ast.IdentifierNode{Value: "other"})
	require.False(t, ok)
	_, ok = pathPart(&ast.BoolNode{Value: true})
	require.False(t, ok)
	_, ok = pathOf(&ast.MemberNode{Node: &ast.IdentifierNode{Value: "this"}, Property: &ast.BoolNode{Value: true}})
	require.False(t, ok)
	require.Nil(t, compileBool("this."))
	floatPredicate := compileBool(`this.x == 1.5`)
	_, _ = floatPredicate.eval([]byte(`{"x":"bad"}`))
	boolPredicate := compileBool(`this.x == true`)
	_, _ = boolPredicate.eval([]byte(`{"x":false}`))
	require.Equal(t, Function{Name: "unknown"}, applyFunctionDoc(Function{Name: "unknown"}))
}

func TestCustomFunctions(t *testing.T) {
	tests := []struct {
		name   string
		source string
		input  string
		want   any
	}{
		{name: "numeric", source: "roundTo(sqrt(this.value), 2)", input: `{"value": 2}`, want: 1.41},
		{name: "collection", source: "cumsum(this.values)", input: `{"values":[1,2,3]}`, want: []any{float64(1), float64(3), float64(6)}},
		{name: "text", source: `regexFind(this.value, "(id-[0-9]+)")`, input: `{"value":"id-42"}`, want: []any{"id-42", "id-42"}},
		{name: "encoding", source: `hash(this.value, "sha256")`, input: `{"value":"hello"}`, want: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{name: "network", source: `normalizeHostname(this.value)`, input: `{"value":"Example.COM."}`, want: "example.com"},
		{name: "validation", source: `is(this.value, "uuid_v4")`, input: `{"value":"550e8400-e29b-41d4-a716-446655440000"}`, want: true},
		{name: "assignment", source: `bucket(this.value, 10) >= 0`, input: `{"value":"hello"}`, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			program, err := Compile(tc.source)
			require.NoError(t, err)
			got, err := evalAt(program, []byte(tc.input), nil, time.Unix(0, 0).UTC())
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestEvaluation(t *testing.T) {
	result := Evaluate(Request{Expression: "now() == now()", At: "2026-08-30T12:00:00+04:00"})
	require.Equal(t, "boolean", result.Type)
	require.Equal(t, true, result.Out)
}

func TestContractMatrix(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		input      []byte
		at         string
		typeName   string
		want       any
	}{
		{name: "null", expression: "nil", typeName: "null", want: nil},
		{name: "boolean", expression: "true", typeName: "boolean", want: true},
		{name: "integer", expression: "42", typeName: "integer", want: int64(42)},
		{name: "number", expression: "1.5", typeName: "number", want: 1.5},
		{name: "string", expression: `"hello"`, typeName: "string", want: "hello"},
		{name: "array", expression: "[1, 2]", typeName: "array", want: []any{int64(1), int64(2)}},
		{name: "object", expression: `{name: "Ada"}`, typeName: "object", want: map[string]any{"name": "Ada"}},
		{name: "time", expression: "time(0)", typeName: "time", want: "1970-01-01T00:00:00Z"},
		{name: "duration", expression: `duration("1s")`, typeName: "duration", want: "1s"},
		{name: "now", expression: "now()", at: "2026-08-30T12:00:00+04:00", typeName: "time", want: "2026-08-30T08:00:00Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Evaluate(Request{Expression: tc.expression, Input: tc.input, At: tc.at})
			require.Equal(t, tc.typeName, result.Type, result.Out)
			got, err := json.Marshal(result.Out)
			require.NoError(t, err)
			want, err := json.Marshal(tc.want)
			require.NoError(t, err)
			assert.JSONEq(t, string(want), string(got))
		})
	}
}

func TestContractLimits(t *testing.T) {
	exactSource := "1" + strings.Repeat(" ", maxSourceBytes-1)
	_, err := Compile(exactSource)
	require.NoError(t, err)
	_, err = Compile(exactSource + " ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source exceeds")

	exactJSON := []byte(`"` + strings.Repeat("x", maxJSONBytes-2) + `"`)
	got, err := validateInput(exactJSON)
	require.NoError(t, err)
	assert.Equal(t, exactJSON, got)
	_, err = validateInput(append(append([]byte(nil), exactJSON...), 'x'))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input exceeds")
	require.NoError(t, Validate(exactJSON))
	err = validateOutput(append(append([]byte(nil), exactJSON...), 'x'))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "output exceeds")
	require.Error(t, Validate([]byte(`{"key":1,"key":2}`)))

	withinJSONDepth := []byte(strings.Repeat("[", maxNesting-1) + "0" + strings.Repeat("]", maxNesting-1))
	require.NoError(t, Validate(withinJSONDepth))
	tooDeepJSON := []byte(strings.Repeat("[", maxNesting) + "0" + strings.Repeat("]", maxNesting))
	err = Validate(tooDeepJSON)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nesting exceeds")

	withinJSONEntries := []byte("[" + strings.Repeat("0,", maxEntries-1) + "0]")
	require.NoError(t, Validate(withinJSONEntries))
	tooManyJSONEntries := []byte("[" + strings.Repeat("0,", maxEntries) + "0]")
	err = Validate(tooManyJSONEntries)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection entries exceed")

	withinDepth := any("leaf")
	for i := 0; i < maxNesting-1; i++ {
		withinDepth = []any{withinDepth}
	}
	_, err = normalizeValue(withinDepth, 1, new(int))
	require.NoError(t, err)
	tooDeep := any("leaf")
	for i := 0; i < maxNesting; i++ {
		tooDeep = []any{tooDeep}
	}
	_, err = normalizeValue(tooDeep, 1, new(int))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nesting exceeds")

	withinEntries := make([]any, maxEntries)
	_, err = normalizeValue(withinEntries, 1, new(int))
	require.NoError(t, err)
	tooWide := make([]any, maxEntries+1)
	_, err = normalizeValue(tooWide, 1, new(int))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection entries exceed")

	program, err := Compile(fmt.Sprintf(`repeat("x", %d)`, maxVMMemory+1))
	require.NoError(t, err)
	_, err = program.Eval(nil, nil, time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "memory budget")
}

func TestEncodedSize(t *testing.T) {
	at := time.Date(2026, time.August, 30, 12, 34, 56, 789, time.UTC)
	cases := []struct {
		name  string
		value any
	}{
		{name: "null", value: nil},
		{name: "boolean", value: true},
		{name: "integer", value: int64(-42)},
		{name: "unsigned", value: uint64(math.MaxUint64)},
		{name: "float32", value: float32(0.1)},
		{name: "number", value: 1.25e-7},
		{name: "escaped string", value: "quote\" slash\\ line\n control\x01 snowman ☃"},
		{name: "time", value: at},
		{name: "duration", value: 5*time.Minute + 3*time.Second},
		{name: "array", value: []any{int64(1), "two", false}},
		{name: "object", value: map[string]any{"when": at, "items": []any{time.Second, "x"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := encodedSize(tc.value, 1, new(int))
			require.NoError(t, err)
			encoded, err := json.Marshal(outputJSONValue(tc.value))
			require.NoError(t, err)
			assert.Equal(t, len(encoded), got)
		})
	}

	_, err := encodedSize("\xff", 1, new(int))
	require.Error(t, err)
	_, err = encodedSize(math.Inf(1), 1, new(int))
	require.Error(t, err)
}

func TestEnvUsage(t *testing.T) {
	cases := []struct {
		source string
		want   envUsage
	}{
		{source: "1"},
		{source: "this.value", want: usesInput},
		{source: "context.value", want: usesContext},
		{source: `json("value")`, want: usesJSON},
		{source: "now()", want: usesNow},
	}
	for _, tc := range cases {
		program, err := Compile(tc.source)
		require.NoError(t, err)
		assert.Equal(t, tc.want, program.uses, tc.source)
	}
}

func TestContextEvaluation(t *testing.T) {
	program, err := Compile(`context["offset"] + this.value`)
	require.NoError(t, err)
	got, err := program.Eval([]byte(`{"offset": 3}`), []byte(`{"value": 2}`), time.Time{})
	require.NoError(t, err)
	assert.Equal(t, int64(5), got)

	_, err = program.Eval([]byte(`{`), []byte(`{"value": 2}`), time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func FuzzCompileNoPanic(f *testing.F) {
	for _, seed := range []string{"", "this.value", `this["a.b"]`, "1 +", `{"value": this.value}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > maxSourceBytes+32 {
			source = source[:maxSourceBytes+32]
		}
		require.NotPanics(t, func() {
			_, _ = Compile(source)
		})
	})
}

func FuzzJSONValidationNoPanic(f *testing.F) {
	for _, seed := range []string{"{}", `{"value":1}`, `"text"`, "{", "NaN"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if len(raw) > maxJSONBytes+32 {
			raw = raw[:maxJSONBytes+32]
		}
		require.NotPanics(t, func() {
			_, _ = validateInput([]byte(raw))
			_ = Validate([]byte(raw))
		})
	})
}

func FuzzProjectionNoPanic(f *testing.F) {
	f.Add("this.value", `{"value":1}`)
	f.Add(`{"value": this.value}`, `{"value":{"nested":true}}`)
	f.Fuzz(func(t *testing.T, source, raw string) {
		if len(source) > 1024 {
			source = source[:1024]
		}
		if len(raw) > 4096 {
			raw = raw[:4096]
		}
		program, err := Compile(source)
		if err != nil {
			return
		}
		require.NotPanics(t, func() {
			_, _ = program.JSON([]byte(raw))
			_, _ = program.AppendJSON([]byte("prefix"), []byte(raw))
		})
	})
}

func TestProgramConcurrency(t *testing.T) {
	program, err := Compile(`{value: this.value * 2, text: trim(this.text)}`)
	require.NoError(t, err)
	for i := 0; i < 16; i++ {
		i := i
		t.Run(fmt.Sprintf("input-%d", i), func(t *testing.T) {
			t.Parallel()
			input := []byte(fmt.Sprintf(`{"value":%d,"text":" value-%d "}`, i, i))
			got, err := program.Eval(nil, input, time.Time{})
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"value": int64(i * 2), "text": fmt.Sprintf("value-%d", i)}, got)
			encoded, err := program.JSON(input)
			require.NoError(t, err)
			assert.JSONEq(t, fmt.Sprintf(`{"value":%d,"text":"value-%d"}`, i*2, i), string(encoded))
		})
	}
}
