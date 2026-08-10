package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOptionalSortOrderQuery(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		raw      string
		expected string
	}{
		{name: "empty", raw: "", expected: ""},
		{name: "ascending", raw: " ASC ", expected: "asc"},
		{name: "descending", raw: "desc", expected: "desc"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := parseOptionalSortOrderQuery(testCase.raw, "content_safety_sort_order")
			require.NoError(t, err)
			require.Equal(t, testCase.expected, actual)
		})
	}

	_, err := parseOptionalSortOrderQuery("desc; drop table users", "content_safety_sort_order")
	require.Error(t, err)
}
