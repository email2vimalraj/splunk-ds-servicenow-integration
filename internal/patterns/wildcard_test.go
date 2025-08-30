package patterns

import "testing"

func TestGenerateWildcards(t *testing.T) {
	hosts := []string{"abc001", "abc002", "xyz101"}
	got := GenerateWildcards(hosts)
	want := []string{"abc*", "xyz101"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d got %q want %q", i, got[i], want[i])
		}
	}
}
