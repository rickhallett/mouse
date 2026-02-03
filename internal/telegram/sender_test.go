package telegram

import "testing"

func TestIsAllowedUser(t *testing.T) {
	user := &User{ID: 42, Username: "tester"}
	if !isAllowedUser([]string{"42"}, user) {
		t.Fatalf("expected allowed by id")
	}
	if !isAllowedUser([]string{"tester"}, user) {
		t.Fatalf("expected allowed by username")
	}
	if isAllowedUser([]string{"other"}, user) {
		t.Fatalf("expected denied")
	}
}
