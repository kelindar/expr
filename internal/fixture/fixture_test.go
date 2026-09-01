// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package fixture

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	engine "github.com/kelindar/expr"
	"github.com/stretchr/testify/require"
)

func TestFixtures(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Dir(filepath.Dir(filepath.Dir(source)))
	groups, err := Files(root)
	require.NoError(t, err)
	for domain, cases := range groups {
		for _, item := range cases {
			t.Run(domain+"/"+item.Name, func(t *testing.T) {
				result := Evaluate(item)
				if item.Error != "" {
					require.Equal(t, "error", result.Type)
					failure, ok := result.Out.(engine.Failure)
					require.True(t, ok)
					require.Equal(t, item.Error, failure.Code)
					return
				}
				require.NotNil(t, item.Expect)
				require.Equalf(t, item.Expect.Type, result.Type, "out=%#v", result.Out)
				got, err := json.Marshal(result.Out)
				require.NoError(t, err)
				want, err := json.Marshal(item.Expect.Out)
				require.NoError(t, err)
				require.JSONEq(t, string(want), string(got))
			})
		}
	}
}

func TestLoadValidation(t *testing.T) {
	for _, data := range []string{
		"not: [yaml",
		"- expression: x\n  expect: {type: string}\n",
		"- name: x\n  expect: {type: string}\n",
		"- name: x\n  expression: x\n",
		"- name: x\n  expression: x\n  expect: {type: string}\n  error: evaluation_error\n",
		"- name: x\n  expression: x\n  expect: {}\n",
	} {
		_, err := Load([]byte(data))
		require.Error(t, err)
	}
}

func TestFixtureHelpers(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "demo", "fixtures.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("- name: demo\n  expression: '1 + 1'\n  expect: {type: integer, out: 2}\n"), 0o600))
	central := filepath.Join(dir, "internal", "fixtures", "core.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(central), 0o755))
	require.NoError(t, os.WriteFile(central, []byte("- name: core\n  expression: '2 + 2'\n  expect: {type: integer, out: 4}\n"), 0o600))
	groups, err := Files(dir)
	require.NoError(t, err)
	require.Len(t, groups["demo"], 1)
	require.Len(t, groups["core"], 1)
	_, err = Files("[")
	require.Error(t, err)
	got, err := InputJSON(Case{})
	require.NoError(t, err)
	require.Nil(t, got)
	got, err = InputJSON(Case{Input: map[string]any{"x": 1}})
	require.NoError(t, err)
	require.JSONEq(t, `{"x":1}`, string(got))
	_, err = InputJSON(Case{Input: func() {}})
	require.Error(t, err)
	result := Evaluate(Case{Name: "bad", Expression: "1 +", Error: "compile_error"})
	require.Equal(t, "error", result.Type)
	result = Evaluate(Case{Name: "input", Expression: "1", Input: func() {}})
	require.Equal(t, "invalid_input", result.Out.(engine.Failure).Code)
	result = Evaluate(Case{Name: "at", Expression: "1", At: "bad"})
	require.Equal(t, "invalid_input", result.Out.(engine.Failure).Code)
}
