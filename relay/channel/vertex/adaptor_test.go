package vertex

import "testing"

func TestVertexClaudeOpus47MappingExists(t *testing.T) {
	t.Parallel()

	got, ok := claudeModelMap["claude-opus-4-7"]
	if !ok {
		t.Fatalf("claude-opus-4-7 missing from Vertex Claude map")
	}
	if got == "" {
		t.Fatalf("claude-opus-4-7 vertex model id should not be empty")
	}
}
