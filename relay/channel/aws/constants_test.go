package aws

import "testing"

func TestAWSClaudeOpus47MappingsExist(t *testing.T) {
	t.Parallel()

	modelID, ok := awsModelIDMap["claude-opus-4-7"]
	if !ok {
		t.Fatalf("claude-opus-4-7 missing from awsModelIDMap")
	}
	if modelID != "anthropic.claude-opus-4-7" {
		t.Fatalf("claude-opus-4-7 aws model id = %q, want %q", modelID, "anthropic.claude-opus-4-7")
	}
	if !awsModelCanCrossRegionMap[modelID]["us"] {
		t.Fatalf("claude-opus-4-7 should support us cross-region routing")
	}
}
