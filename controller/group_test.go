package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortedGroupNamesIsStable(t *testing.T) {
	groupRatios := map[string]float64{
		"MiMo · 优质":        1,
		"Claude · 企业专属":    2,
		"OpenAI · 优质（第三方）": 0.45,
		"default":          1,
	}

	require.Equal(t, []string{
		"Claude · 企业专属",
		"MiMo · 优质",
		"OpenAI · 优质（第三方）",
		"default",
	}, sortedGroupNames(groupRatios))
}
