package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBool(t *testing.T) {
	tests := map[string]struct {
		source  string
		input   string
		want    bool
		wantErr bool
	}{
		"matches fast path": {
			source: `this.ready == true`,
			input:  `{"ready":true}`,
			want:   true,
		},
		"evaluates false": {
			source: `this.count >= 2`,
			input:  `{"count":1}`,
			want:   false,
		},
		"matches reversed fast path": {
			source: `1 <= this.count`,
			input:  `{"count":2}`,
			want:   true,
		},
		"rejects nested path": {
			source:  `this.a.b == nil`,
			input:   `{}`,
			wantErr: true,
		},
		"missing context path": {
			source: `context.ready == true`,
			input:  `{"ready":true}`,
			want:   false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			program, err := Compile(tc.source)
			require.NoError(t, err)

			got, err := Bool(program, []byte(tc.input))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
