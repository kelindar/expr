package expr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuiltins(t *testing.T) {
	t.Run("getFunc", func(t *testing.T) {
		root := value{raw: []byte(`{"items":[{"state":"failed","id":"b"}]}`)}
		get := getFunc(root)

		require.Equal(t, "b", get("items.0.id"))
		require.Nil(t, get("items.9.id"))
	})

	t.Run("jsonRootSelector", func(t *testing.T) {
		root := value{raw: []byte(`{"items":[{"state":"done","id":"a"},{"state":"failed","id":"b"}]}`)}
		fn := jsonFunc(root)

		got, err := fn(`items.#(state=="failed").id`)
		require.NoError(t, err)
		require.Equal(t, "b", got)
	})

	t.Run("jsonEmbedded", func(t *testing.T) {
		root := value{raw: []byte(`{"raw":"{\"name\":\"Roman\",\"age\":12}"}`)}
		fn := jsonFunc(root)

		got, err := fn(root.Get("raw"), "age")
		require.NoError(t, err)
		require.Equal(t, int64(12), got)
	})

	t.Run("jsonErrors", func(t *testing.T) {
		root := value{raw: []byte(`{"state":"done"}`)}
		fn := jsonFunc(root)

		_, err := fn()
		require.Error(t, err)
		_, err = fn(12, "state")
		require.Error(t, err)
		_, err = fn("state", 12)
		require.Error(t, err)
	})

	t.Run("jsonRaw", func(t *testing.T) {
		root := value{raw: []byte(`{"payload":{"name":"ada"}}`)}
		raw, err := jsonRaw(value{raw: root.raw, path: "payload"})
		require.NoError(t, err)
		require.JSONEq(t, `{"name":"ada"}`, string(raw))

		_, err = jsonRaw(value{raw: root.raw, path: "missing"})
		require.Error(t, err)
	})

	t.Run("fnTime", func(t *testing.T) {
		seconds, err := fnTime(int64(1_577_836_800))
		require.NoError(t, err)
		require.Equal(t, time.Unix(1_577_836_800, 0), seconds)

		nanos, err := fnTime(int64(1_716_336_000_123_456_789))
		require.NoError(t, err)
		require.Equal(t, time.Unix(0, 1_716_336_000_123_456_789), nanos)

		_, err = fnTime("not-a-number")
		require.Error(t, err)
	})

	t.Run("fnDuration", func(t *testing.T) {
		got, err := fnDuration("5m30s")
		require.NoError(t, err)
		require.Equal(t, 5*time.Minute+30*time.Second, got)

		_, err = fnDuration("not-a-duration")
		require.Error(t, err)
	})

	t.Run("number", func(t *testing.T) {
		n, ok := number(42)
		require.True(t, ok)
		require.Equal(t, int64(42), n)

		n, ok = number(float64(12))
		require.True(t, ok)
		require.Equal(t, int64(12), n)

		n, ok = number("99")
		require.True(t, ok)
		require.Equal(t, int64(99), n)

		_, ok = number(true)
		require.False(t, ok)
	})
}
