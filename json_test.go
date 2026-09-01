package expr

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type jsonFixture struct {
	Name   string `yaml:"name"`
	Expr   string `yaml:"expr"`
	Input  string `yaml:"input"`
	Expect string `yaml:"expect"`
	Fast   bool   `yaml:"fast"`
	Error  bool   `yaml:"error"`
}

func TestJSON(t *testing.T) {
	for _, tc := range readJSONFixtures(t) {
		t.Run(tc.Name, func(t *testing.T) {
			program, err := Compile(tc.Expr)
			require.NoError(t, err)
			require.Equal(t, tc.Fast, program.json != nil)

			got, err := JSON(program, []byte(tc.Input))
			if tc.Error {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.Expect, string(got))
		})
	}
}

func readJSONFixtures(t *testing.T) []jsonFixture {
	t.Helper()
	return readYAMLFile[jsonFixture](t, filepath.Join("internal", "fixtures", "expr", "json.yaml"))
}
