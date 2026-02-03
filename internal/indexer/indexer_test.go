package indexer

import "testing"

func TestTokenize(t *testing.T) {
	got := tokenize("Hello, world! hello 123")
	if len(got) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(got))
	}
	if got[0] != "hello" {
		t.Fatalf("unexpected token[0]: %q", got[0])
	}
}

func TestSimilarity(t *testing.T) {
	score := similarity([]string{"alpha", "beta"}, []string{"beta", "gamma"})
	if score <= 0 {
		t.Fatalf("expected positive score")
	}
}
