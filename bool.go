package expr

import (
	"cmp"
	"strconv"
	"strings"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/tidwall/gjson"
)

type boolPredicate struct {
	path, op string
	kind     byte
	integer  int64
	number   float64
	text     string
	boolean  bool
}

func compileBool(source string) *boolPredicate {
	tree, err := parser.Parse(source)
	var binary *ast.BinaryNode
	var binaryOK bool
	if err == nil {
		if node, ok := tree.Node.(*ast.BinaryNode); ok && comparison(node.Operator) {
			binary, binaryOK = node, true
		}
	}
	switch {
	case err != nil:
		return nil
	case !binaryOK:
		return nil
	}
	path, ok := pathOf(binary.Left)
	if ok && strings.Contains(path, ".") {
		return nil
	}
	value, literal := predicateLiteral(binary.Right)
	if ok && literal && value.supports(binary.Operator) {
		value.path, value.op = path, binary.Operator
		return value
	}
	path, ok = pathOf(binary.Right)
	if !ok || strings.Contains(path, ".") {
		return nil
	}
	value, literal = predicateLiteral(binary.Left)
	if !ok || !literal || !value.supports(binary.Operator) {
		return nil
	}
	value.path, value.op = path, reverseComparison(binary.Operator)
	return value
}

func (p *boolPredicate) supports(op string) bool {
	return (p.kind != 'n' && p.kind != 'b') || op == "==" || op == "!="
}

func predicateLiteral(node ast.Node) (*boolPredicate, bool) {
	switch value := node.(type) {
	case *ast.NilNode:
		return &boolPredicate{kind: 'n'}, true
	case *ast.IntegerNode:
		return &boolPredicate{kind: 'i', integer: int64(value.Value)}, true
	case *ast.FloatNode:
		return &boolPredicate{kind: 'f', number: value.Value}, true
	case *ast.StringNode:
		return &boolPredicate{kind: 's', text: value.Value}, true
	case *ast.BoolNode:
		return &boolPredicate{kind: 'b', boolean: value.Value}, true
	default:
		return nil, false
	}
}

func (p *boolPredicate) eval(input []byte) (bool, bool) {
	value := parseJSON(input)
	if p.path != "" {
		value = getJSON(input, p.path)
	}
	var order int
	switch p.kind {
	case 'n':
		if value.Exists() && value.Type != gjson.Null {
			return false, false
		}
	case 'i':
		actual, err := strconv.ParseInt(value.Raw, 10, 64)
		if err != nil {
			return false, false
		}
		order = cmp.Compare(actual, p.integer)
	case 'f':
		actual, err := strconv.ParseFloat(value.Raw, 64)
		if err != nil {
			return false, false
		}
		order = cmp.Compare(actual, p.number)
	case 's':
		if value.Type != gjson.String {
			return false, false
		}
		order = cmp.Compare(value.String(), p.text)
	case 'b':
		if value.Type != gjson.True && value.Type != gjson.False {
			return false, false
		}
		if value.Bool() != p.boolean {
			order = 1
		}
	}
	return compareOrder(order, p.op), true
}

func comparison(op string) bool {
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}

func reverseComparison(op string) string {
	switch op {
	case "<":
		return ">"
	case "<=":
		return ">="
	case ">":
		return "<"
	case ">=":
		return "<="
	default:
		return op
	}
}

func compareOrder(order int, op string) bool {
	switch op {
	case "==":
		return order == 0
	case "!=":
		return order != 0
	case "<":
		return order < 0
	case "<=":
		return order <= 0
	case ">":
		return order > 0
	default:
		return order >= 0
	}
}
