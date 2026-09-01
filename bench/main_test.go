// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root.

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/kelindar/expr/internal/fixture"
	"github.com/stretchr/testify/require"
)

func TestBenchmarkCommand(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(source))
	err := run(root, 2, time.Nanosecond)
	require.NoError(t, err)

	customRoot := t.TempDir()
	validDir := filepath.Join(customRoot, "custom")
	require.NoError(t, os.MkdirAll(validDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(validDir, "fixtures.yaml"), []byte(`
- name: invalid compile
  expression: "this."
  expect:
    type: string
  benchmark: [invalid]
- name: invalid input
  expression: "1"
  input: {value: .nan}
  expect:
    type: integer
  benchmark: [invalid]
`), 0o644))
	require.NoError(t, run(customRoot, 1, time.Nanosecond))

	badRoot := t.TempDir()
	badDir := filepath.Join(badRoot, "broken")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, "fixtures.yaml"), []byte("["), 0o644))
	require.Error(t, run(badRoot, 1, time.Nanosecond))

	oldRoot, oldSamples, oldDuration := benchmarkRoot, benchmarkSamples, benchmarkDuration
	defer func() {
		benchmarkRoot, benchmarkSamples, benchmarkDuration = oldRoot, oldSamples, oldDuration
	}()
	benchmarkRoot, benchmarkSamples, benchmarkDuration = "", 1, time.Nanosecond
	main()
	benchmarkRoot = badRoot
	require.Panics(t, main)
}

func TestBenchmarkNames(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(source))
	groups, err := fixture.Files(root)
	require.NoError(t, err)
	collisions := benchmarkCollisions(groups)
	require.NotEmpty(t, largestBenchmarkExpression(groups))
	seen := make(map[string]struct{})
	seen["compile/large"] = struct{}{}
	for domain, cases := range groups {
		for _, item := range cases {
			if len(item.Benchmark) == 0 {
				continue
			}
			full := benchmarkName(domain, item.Name, collisions)
			require.Less(t, len(full), 20, full)
			require.NotContains(t, seen, full)
			seen[full] = struct{}{}
		}
	}
}
