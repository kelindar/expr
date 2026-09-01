package expr

import (
	"testing"

	"github.com/expr-lang/expr/ast"
	"github.com/stretchr/testify/require"
)

func TestPathOf(t *testing.T) {
	t.Run("this", func(t *testing.T) {
		path, ok := pathOf(&ast.IdentifierNode{Value: "this"})
		require.True(t, ok)
		require.Empty(t, path)
	})

	t.Run("memberChain", func(t *testing.T) {
		this := &ast.IdentifierNode{Value: "this"}
		state := &ast.MemberNode{
			Node:     this,
			Property: &ast.StringNode{Value: "state"},
		}

		path, ok := pathOf(state)
		require.True(t, ok)
		require.Equal(t, "state", path)
	})

	t.Run("nestedMembers", func(t *testing.T) {
		this := &ast.IdentifierNode{Value: "this"}
		user := &ast.MemberNode{
			Node:     this,
			Property: &ast.StringNode{Value: "user"},
		}
		profile := &ast.MemberNode{
			Node:     user,
			Property: &ast.StringNode{Value: "profile"},
		}
		age := &ast.MemberNode{
			Node:     profile,
			Property: &ast.StringNode{Value: "age"},
		}

		path, ok := pathOf(age)
		require.True(t, ok)
		require.Equal(t, "user.profile.age", path)
	})

	t.Run("arrayIndex", func(t *testing.T) {
		this := &ast.IdentifierNode{Value: "this"}
		items := &ast.MemberNode{
			Node:     this,
			Property: &ast.StringNode{Value: "items"},
		}
		first := &ast.MemberNode{
			Node:     items,
			Property: &ast.IntegerNode{Value: 0},
		}

		path, ok := pathOf(first)
		require.True(t, ok)
		require.Equal(t, "items.0", path)
	})

	t.Run("rejectsOptionalAccess", func(t *testing.T) {
		this := &ast.IdentifierNode{Value: "this"}
		optional := &ast.MemberNode{
			Node:     this,
			Property: &ast.StringNode{Value: "state"},
			Optional: true,
		}
		method := &ast.MemberNode{
			Node:     this,
			Property: &ast.StringNode{Value: "trim"},
			Method:   true,
		}

		_, ok := pathOf(optional)
		require.False(t, ok)
		_, ok = pathOf(method)
		require.False(t, ok)
	})
}

func TestEscapePathPart(t *testing.T) {
	require.Equal(t, "state", escapePathPart("state"))
	require.Equal(t, `weird\.key`, escapePathPart("weird.key"))
	require.Equal(t, `\*`, escapePathPart("*"))
	require.Equal(t, `\?`, escapePathPart("?"))
	require.Equal(t, `a\\b`, escapePathPart(`a\b`))
	require.Equal(t, "plain-name", escapePathPart("plain-name"))
}

func TestPatcher(t *testing.T) {
	program, err := Compile(`this.user.profile.age > 18`)
	require.NoError(t, err)
	require.NotNil(t, program)

	got, err := Eval(program, []byte(`{"user":{"profile":{"age":42}}}`))
	require.NoError(t, err)
	require.Equal(t, true, got)
}

func TestPatcherSpecialKeys(t *testing.T) {
	program, err := Compile(`this["a.b"] + this["*"] + this["?"]`)
	require.NoError(t, err)
	got, err := Eval(program, []byte(`{"a.b":1,"*":2,"?":3}`))
	require.NoError(t, err)
	require.Equal(t, int64(6), got)
}
