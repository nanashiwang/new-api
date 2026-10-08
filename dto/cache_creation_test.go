package dto

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveCacheCreation(t *testing.T) {
	for _, tc := range []struct {
		total, five, hour int
		reported          bool
		want              CacheCreation
		invalid           bool
	}{
		{400, 80, 120, true, CacheCreation{400, 80, 120, 200}, false},
		{400, 0, 0, true, CacheCreation{400, 0, 0, 400}, false},
		{0, 80, 120, false, CacheCreation{200, 80, 120, 0}, false},
		{0, 0, 0, true, CacheCreation{}, false},
		{0, 0, 120, true, CacheCreation{}, true},
		{80, 80, 120, true, CacheCreation{}, true},
		{-1, 0, 0, true, CacheCreation{}, true},
		{100, -1, 0, true, CacheCreation{}, true},
		{100, 0, -1, true, CacheCreation{}, true},
		{0, math.MaxInt, 1, false, CacheCreation{}, true},
	} {
		got, err := ResolveCacheCreation(tc.total, tc.five, tc.hour, tc.reported)
		if tc.invalid {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		}
	}
}
