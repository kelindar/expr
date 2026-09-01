// Package fixture owns the shared YAML format for expression examples.
package fixture

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	engine "github.com/kelindar/expr"
	"gopkg.in/yaml.v3"
)

// Case is one expression fixture. Exactly one of Expect or Error must be set.
type Case struct {
	Name       string       `yaml:"name"`
	Expression string       `yaml:"expression"`
	Input      any          `yaml:"input,omitempty"`
	At         string       `yaml:"at,omitempty"`
	Expect     *Expectation `yaml:"expect,omitempty"`
	Error      string       `yaml:"error,omitempty"`
	Benchmark  []string     `yaml:"benchmark,omitempty"`
}

// Expectation describes the stable typed result envelope.
type Expectation struct {
	Type string `yaml:"type"`
	Out  any    `yaml:"out,omitempty"`
}

// Load decodes and validates expression fixtures from YAML.
func Load(data []byte) ([]Case, error) {
	var cases []Case
	if err := yaml.Unmarshal(data, &cases); err != nil {
		return nil, err
	}
	for i := range cases {
		item := &cases[i]
		switch {
		case item.Name == "" || item.Expression == "":
			return nil, fmt.Errorf("fixture %d: name and expression are required", i)
		case (item.Expect == nil) == (item.Error == ""):
			return nil, fmt.Errorf("fixture %q: set exactly one of expect or error", item.Name)
		case item.Expect != nil && item.Expect.Type == "":
			return nil, fmt.Errorf("fixture %q: expected type is required", item.Name)
		}
	}
	return cases, nil
}

// LoadFile reads and decodes one fixture file.
func LoadFile(path string) ([]Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Load(data)
}

// Files loads all domain fixture files below root in stable order. Domain files
// may live beside their package or in internal/fixtures.
func Files(root string) (map[string][]Case, error) {
	var paths []string
	for _, pattern := range []string{
		filepath.Join(root, "*", "fixtures.yaml"),
		filepath.Join(root, "internal", "fixtures", "*.yaml"),
	} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		paths = append(paths, matches...)
	}
	slices.Sort(paths)
	result := make(map[string][]Case, len(paths))
	for _, path := range paths {
		cases, err := LoadFile(path)
		if err != nil {
			return nil, err
		}
		domain := filepath.Base(filepath.Dir(path))
		if domain == "fixtures" {
			domain = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		result[domain] = cases
	}
	return result, nil
}

// InputJSON returns the fixture input in the evaluator's JSON form.
func InputJSON(item Case) ([]byte, error) {
	if item.Input == nil {
		return nil, nil
	}
	return json.Marshal(item.Input)
}

// Evaluate executes a fixture using the production typed contract.
func Evaluate(item Case) engine.Result {
	input, err := InputJSON(item)
	if err != nil {
		return engine.Result{Type: "error", Out: engine.Failure{Code: "invalid_input", Message: err.Error()}}
	}
	return engine.Evaluate(engine.Request{Expression: item.Expression, Input: input, At: item.At})
}
