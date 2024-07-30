package hook

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_checkIsJSON(t *testing.T) {
	msgBz := []byte(`{"is_json": true, "hello": "world"}`)
	isJSON, msgBz, err := checkIsJSON(msgBz)
	require.NoError(t, err)
	require.True(t, isJSON)
	require.Equal(t, []byte(`{"hello":"world"}`), msgBz)

	msgBz = []byte(`{"is_json": false, "hello": "world"}`)
	isJSON, msgBz, err = checkIsJSON(msgBz)
	require.NoError(t, err)
	require.False(t, isJSON)
	require.Equal(t, []byte(`{"hello":"world"}`), msgBz)

	msgBz = []byte(`{"hello": "world"}`)
	isJSON, msgBz, err = checkIsJSON(msgBz)
	require.NoError(t, err)
	require.False(t, isJSON)
	require.Equal(t, []byte(`{"hello":"world"}`), msgBz)
}
