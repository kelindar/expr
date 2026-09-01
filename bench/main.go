// Command bench runs the expression fixture benchmarks.
package main

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/kelindar/bench"
	engine "github.com/kelindar/expr"
	"github.com/kelindar/expr/internal/fixture"
)

var (
	benchmarkRoot     string
	benchmarkSamples  = 20
	benchmarkDuration = 200 * time.Millisecond
	benchmarkProgram  *engine.Program
	benchmarkValue    any
	benchmarkErr      error
)

func main() {
	root := benchmarkRoot
	if root == "" {
		_, source, _, _ := runtime.Caller(0)
		root = filepath.Dir(filepath.Dir(source))
	}
	if err := run(root, benchmarkSamples, benchmarkDuration); err != nil {
		panic(err)
	}
}

func run(root string, samples int, duration time.Duration) error {
	groups, err := fixture.Files(root)
	if err != nil {
		return err
	}
	collisions := benchmarkCollisions(groups)
	large := largestBenchmarkExpression(groups)
	runBenchmarks(groups, collisions, large, samples, duration)
	return nil
}

func runBenchmarks(groups map[string][]fixture.Case, collisions map[string]bool, large string, samples int, duration time.Duration) {
	bench.Run(func(runner *bench.B) {
		if large != "" {
			runner.Run("compile/large", func(_ int) {
				benchmarkProgram, benchmarkErr = engine.Compile(large)
			})
		}
		domains := make([]string, 0, len(groups))
		for domain := range groups {
			domains = append(domains, domain)
		}
		sort.Strings(domains)
		for _, domain := range domains {
			cases := groups[domain]
			sort.Slice(cases, func(i, j int) bool { return cases[i].Name < cases[j].Name })
			benchmarkCases(runner, domain, cases, collisions)
		}
	}, bench.WithSamples(samples), bench.WithDuration(duration), bench.WithDryRun())
}

func benchmarkCases(runner *bench.B, domain string, cases []fixture.Case, collisions map[string]bool) {
	for _, item := range cases {
		if len(item.Benchmark) == 0 {
			continue
		}
		program, compileErr := engine.Compile(item.Expression)
		if compileErr != nil {
			continue
		}
		input, inputErr := fixture.InputJSON(item)
		if inputErr != nil {
			continue
		}
		runner.Run(benchmarkName(domain, item.Name, collisions), func(_ int) {
			benchmarkValue, benchmarkErr = engine.Eval(program, input)
		})
	}
}

func largestBenchmarkExpression(groups map[string][]fixture.Case) string {
	large := ""
	for _, cases := range groups {
		for _, item := range cases {
			if len(item.Benchmark) == 0 || len(item.Expression) < len(large) ||
				(len(item.Expression) == len(large) && item.Expression <= large) {
				continue
			}
			if _, err := engine.Compile(item.Expression); err == nil {
				large = item.Expression
			}
		}
	}
	return large
}

func benchmarkCollisions(groups map[string][]fixture.Case) map[string]bool {
	counts := make(map[string]int)
	for _, cases := range groups {
		for _, item := range cases {
			if len(item.Benchmark) > 0 {
				counts[item.Name]++
			}
		}
	}
	collisions := make(map[string]bool)
	for name, count := range counts {
		collisions[name] = count > 1
	}
	return collisions
}

func benchmarkName(domain, name string, collisions map[string]bool) string {
	if collisions[name] {
		name = shortDomain(domain) + "/" + name
	}
	switch name {
	case "rfc3339-without-zone":
		name = "rfc3339-nozone"
	case "removeDiacritics":
		name = "removeDia"
	case "printable-ascii":
		name = "printable"
	default:
		if strings.HasPrefix(name, "normalize") && len(name) > len("normalize") {
			name = "norm" + strings.TrimPrefix(name, "normalize")
		}
	}
	return name
}

func shortDomain(domain string) string {
	switch domain {
	case "upstream":
		return "up"
	case "validate":
		return "val"
	default:
		return domain
	}
}
