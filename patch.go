package expr

import (
	"strconv"
	"strings"

	"github.com/expr-lang/expr/ast"
)

type patcher struct{}

// Visit rewrites a supported member path to the bounded JSON projection helper.
func (patcher) Visit(node *ast.Node) {
	path, root, ok := expressionPath(*node)
	if !ok || path == "" {
		return
	}
	getter := "__get"
	if root == "context" {
		getter = "__context_get"
	}
	ast.Patch(node, &ast.CallNode{
		Callee: &ast.IdentifierNode{Value: getter},
		Arguments: []ast.Node{
			&ast.StringNode{Value: path},
		},
	})
}

func pathOf(node ast.Node) (string, bool) {
	path, root, ok := expressionPath(node)
	return path, ok && root == "this"
}

func expressionPath(node ast.Node) (string, string, bool) {
	switch n := node.(type) {
	case *ast.IdentifierNode:
		return "", n.Value, n.Value == "this" || n.Value == "context"
	case *ast.MemberNode:
		if n.Optional || n.Method {
			return "", "", false
		}
		base, root, ok := expressionPath(n.Node)
		if !ok {
			return "", "", false
		}
		part, ok := pathPart(n.Property)
		if !ok {
			return "", "", false
		}
		if base == "" {
			return part, root, true
		}
		return base + "." + part, root, true
	default:
		return "", "", false
	}
}

func pathPart(node ast.Node) (string, bool) {
	switch n := node.(type) {
	case *ast.StringNode:
		return escapePathPart(n.Value), true
	case *ast.IntegerNode:
		return strconv.Itoa(n.Value), true
	default:
		return "", false
	}
}

func escapePathPart(part string) string {
	if !strings.ContainsAny(part, `\\.|@!*?[]{}#`) {
		return part
	}
	var out strings.Builder
	out.Grow(len(part) + strings.Count(part, "\\"))
	for _, char := range part {
		if strings.ContainsRune(`\\.|@!*?[]{}#`, char) {
			out.WriteByte('\\')
		}
		out.WriteRune(char)
	}
	return out.String()
}
