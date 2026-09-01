// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package expr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestValue(t *testing.T) {
	t.Run("lookup", func(t *testing.T) {
		raw := []byte(`{"state":"failed","count":42,"ratio":1.5,"ok":true,"missing":null,"items":[{"id":"a"}]}`)
		root := value{raw: raw}

		require.Equal(t, "failed", root.Get("state"))
		require.Equal(t, int64(42), root.Get("count"))
		require.Equal(t, 1.5, root.Get("ratio"))
		require.Equal(t, true, root.Get("ok"))
		require.Nil(t, root.Get("missing"))
		require.Nil(t, root.Get("absent"))

		items := root.Get("items").([]any)
		require.Len(t, items, 1)
		require.Equal(t, map[string]any{"id": "a"}, items[0])
	})

	t.Run("index", func(t *testing.T) {
		raw := []byte(`{"items":["first","second"]}`)
		root := value{raw: raw, path: "items"}

		require.Equal(t, "first", root.Index(0))
		require.Equal(t, "second", root.Index(1))
	})

	t.Run("decode numbers", func(t *testing.T) {
		root := resultValue(gjson.Parse(`{"n":42,"nested":[3.5]}`)).(map[string]any)
		require.Equal(t, int64(42), root["n"])
		require.Equal(t, 3.5, root["nested"].([]any)[0])
	})

	t.Run("unwrap", func(t *testing.T) {
		raw := []byte(`{"name":"ada"}`)
		require.Equal(t, map[string]any{"name": "ada"}, unwrap(value{raw: raw}))
		require.Equal(t, "ada", unwrap(value{raw: raw, path: "name"}))
	})
}

func TestEval(t *testing.T) {
	t.Run("requiresValidJSON", func(t *testing.T) {
		program, err := Compile(`this.state == "done"`)
		require.NoError(t, err)

		_, err = program.Eval(nil, []byte(`{not-json`), time.Time{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "valid json")
	})

	t.Run("returnsNonBool", func(t *testing.T) {
		program, err := Compile(`this.state`)
		require.NoError(t, err)

		got, err := program.Eval(nil, []byte(`{"state":"done"}`), time.Time{})
		require.NoError(t, err)
		require.Equal(t, "done", got)
	})
}

func TestFromResult(t *testing.T) {
	raw := []byte(`{"createdAt":1716336000123456789}`)
	result := gjson.GetBytes(raw, "createdAt")
	require.Equal(t, int64(1716336000123456789), fromResult(result, raw, "createdAt"))
}
