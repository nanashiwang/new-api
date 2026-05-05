package codex

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestModelListIncludesHiddenCodexAutoReview(t *testing.T) {
	found := false
	for _, model := range ModelList {
		if model == constant.CodexAutoReviewModel {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ModelList should include %q", constant.CodexAutoReviewModel)
	}
	if !constant.IsHiddenModel(constant.CodexAutoReviewModel) {
		t.Fatalf("%q should be marked hidden", constant.CodexAutoReviewModel)
	}
}
