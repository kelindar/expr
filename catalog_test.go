package expr_test

import (
	"path/filepath"
	"runtime"
	"testing"

	engine "github.com/kelindar/expr"
	"github.com/kelindar/expr/internal/fixture"
	"github.com/kelindar/expr/validate"
	"github.com/stretchr/testify/require"
)

func TestCatalogSource(t *testing.T) {
	require.NotEmpty(t, engine.Functions())
	require.NotEmpty(t, engine.Guide())
}

// TestCatalogCoverage keeps the documented custom surface, validation rules,
// fixtures, and benchmark metadata in lockstep.
func TestCatalogCoverage(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Dir(source)
	groups, err := fixture.Files(root)
	require.NoError(t, err)

	catalog := make(map[string]int)
	for _, item := range engine.Functions() {
		catalog[item.Name]++
		require.NotEmpty(t, item.Description, item.Name)
		require.NotEmpty(t, item.Usage, item.Name)
		require.NotEmpty(t, item.Returns, item.Name)
		require.NotEmpty(t, item.Example, item.Name)
		if item.Domain != "expr" {
			_, err := engine.Compile(item.Example)
			require.NoError(t, err, item.Name)
		}
	}

	benchmarks := make(map[string]int)
	rules := make(map[string]struct{}, len(validate.Rules()))
	for _, rule := range validate.Rules() {
		rules[rule.Name] = struct{}{}
	}
	for _, cases := range groups {
		for _, item := range cases {
			for _, name := range item.Benchmark {
				benchmarks[name]++
				_, functionOK := catalog[name]
				_, ruleOK := rules[name]
				require.True(t, functionOK || ruleOK, "benchmark %q has no catalog function or validation rule", name)
			}
		}
	}
	for name := range catalog {
		require.Positive(t, benchmarks[name], "catalog function %q has no benchmark metadata", name)
	}
	for _, rule := range validate.Rules() {
		require.Positive(t, benchmarks[rule.Name], "validation rule %q has no fixture benchmark", rule.Name)
	}
}
