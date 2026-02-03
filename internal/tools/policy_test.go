package tools

import "testing"

func TestPolicy(t *testing.T) {
	policy := NewPolicy([]string{"read", "write"}, []string{"exec"})
	if !policy.Allowed("read") {
		t.Fatalf("expected read allowed")
	}
	if policy.Allowed("exec") {
		t.Fatalf("expected exec denied")
	}
}
