package expr

import (
	"cmp"
	"encoding/json/v2"
	"errors"
	"slices"
	"strings"
	"unsafe"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/tidwall/gjson"
)

var errJSONFallback = errors.New("expression: json fast path unavailable")

type jsonProjection []jsonField

type jsonField struct {
	key, path string
	literal   []byte
	trim      bool
}

func compileJSON(source string) jsonProjection {
	tree, err := parser.Parse(source)
	var object *ast.MapNode
	var objectOK bool
	if err == nil {
		object, objectOK = tree.Node.(*ast.MapNode)
	}
	switch {
	case err != nil:
		return nil
	case !objectOK:
		return nil
	}
	fields := make(jsonProjection, 0, len(object.Pairs))
	seen := make(map[string]struct{}, len(object.Pairs))
	for _, node := range object.Pairs {
		pair, pairOK := node.(*ast.PairNode)
		var key *ast.StringNode
		keyOK := false
		if pairOK {
			key, keyOK = pair.Key.(*ast.StringNode)
		}
		if !pairOK || !keyOK {
			return nil
		}
		if _, ok := seen[key.Value]; ok {
			return nil
		}
		seen[key.Value] = struct{}{}
		field, ok := compileJSONField(key.Value, pair.Value)
		if !ok {
			return nil
		}
		fields = append(fields, field)
	}
	slices.SortFunc(fields, func(a, b jsonField) int { return cmp.Compare(a.key, b.key) })
	for i := range fields {
		encoded, _ := json.Marshal(fields[i].key)
		fields[i].key = string(encoded)
	}
	return fields
}

func compileJSONField(key string, node ast.Node) (jsonField, bool) {
	field := jsonField{key: key}
	if path, ok := jsonPath(node); ok {
		field.path = path
		return field, true
	}
	if call, ok := node.(*ast.BuiltinNode); ok && call.Name == "trim" && len(call.Arguments) == 1 {
		path, ok := jsonPath(call.Arguments[0])
		if !ok {
			return jsonField{}, false
		}
		field.path, field.trim = path, true
		return field, true
	}
	var value any
	switch node := node.(type) {
	case *ast.NilNode:
		field.literal = []byte("null")
		return field, true
	case *ast.StringNode:
		value = node.Value
	case *ast.IntegerNode:
		value = node.Value
	case *ast.FloatNode:
		value = node.Value
	case *ast.BoolNode:
		value = node.Value
	default:
		return jsonField{}, false
	}
	field.literal, _ = json.Marshal(value)
	return field, true
}

func jsonKeys(node ast.Node) []string {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		if n.Value == "this" {
			return []string{}
		}
	case *ast.MemberNode:
		keys := jsonKeys(n.Node)
		if keys == nil || n.Optional || n.Method {
			return nil
		}
		if key, ok := n.Property.(*ast.StringNode); ok {
			return append(keys, key.Value)
		}
	}
	return nil
}

func jsonPath(node ast.Node) (string, bool) {
	if id, ok := node.(*ast.IdentifierNode); ok && id.Value == "this" {
		return "", true
	}
	path, ok := pathOf(node)
	return path, ok && path != ""
}

func (p jsonProjection) evalValid(input []byte) ([]byte, error) {
	out := make([]byte, 0, len(input)+len(p)*8)
	return p.appendValid(out, input)
}

func (p jsonProjection) appendValid(out, input []byte) ([]byte, error) {
	start := len(out)
	out = append(out, '{')
	for i, field := range p {
		if i > 0 {
			out = append(out, ',')
			if err := projectionOutputLimit(out, start); err != nil {
				return nil, err
			}
		}
		out = append(out, field.key...)
		out = append(out, ':')
		out = field.appendValue(out, input)
		if out == nil {
			return nil, errJSONFallback
		}
		if err := projectionOutputLimit(out, start); err != nil {
			return nil, err
		}
	}
	out = append(out, '}')
	if err := projectionOutputLimit(out, start); err != nil {
		return nil, err
	}
	return out, nil
}

func projectionOutputLimit(out []byte, start int) error {
	if len(out)-start > maxJSONBytes {
		return outputLimitError()
	}
	return nil
}

func (f jsonField) appendValue(out, input []byte) []byte {
	switch {
	case f.literal != nil:
		return append(out, f.literal...)
	case f.path == "" && len(input) == 0:
		return append(out, "null"...)
	}
	return f.appendGJSONValue(out, input)
}

func (f jsonField) appendGJSONValue(out, input []byte) []byte {
	var value gjson.Result
	raw := input
	if f.path != "" {
		value = getJSON(input, f.path)
		if !value.Exists() {
			if f.trim {
				return nil
			}
			return append(out, "null"...)
		}
		raw = []byte(value.Raw)
	}
	if f.path == "" && f.trim {
		value = parseJSON(input)
	}
	switch {
	case !f.trim:
		return append(out, raw...)
	case value.Type != gjson.String:
		return nil
	}
	encoded, _ := json.Marshal(strings.TrimSpace(value.String()))
	return append(out, encoded...)
}

func parseJSON(input []byte) gjson.Result {
	return gjson.Parse(unsafe.String(unsafe.SliceData(input), len(input)))
}

func getJSON(input []byte, path string) gjson.Result {
	return gjson.Get(unsafe.String(unsafe.SliceData(input), len(input)), path)
}
