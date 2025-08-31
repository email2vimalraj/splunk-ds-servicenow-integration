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

func TestGenerateWildcards_InternalNumeric_Grouping(t *testing.T) {
	hosts := []string{
		"api01-us-east-001",
		"api02-us-east-001",
		"api03-us-east-001",
	}
	got := GenerateWildcardsWithOptions(hosts, Options{Mode: "internalNumeric", MinGroupSize: 2})
	want := []string{"api*-us-east-*"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGenerateWildcards_InternalNumeric_ThresholdAndFallback(t *testing.T) {
	hosts := []string{
		"web-a-01", "web-a-02", // 2 hosts only
		"web-b-01", "web-b-02", "web-b-03", // 3 hosts
	}
	got := GenerateWildcardsWithOptions(hosts, Options{Mode: "internalNumeric", MinGroupSize: 3})
	// Expect explicit for web-a, wildcard for web-b
	// Order deterministic after sort
	want := []string{"web-a-01", "web-a-02", "web-b-*"}
	if len(got) != len(want) {
		t.Fatalf("len got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("idx %d got %q want %q", i, got[i], want[i])
		}
	}
}

func TestGenerateWildcards_InternalNumeric_MinFixedPrefix(t *testing.T) {
	hosts := []string{"a1", "a2", "a3"}
	// If we require a longer fixed prefix, should fallback to explicit + trailing
	got := GenerateWildcardsWithOptions(hosts, Options{Mode: "internalNumeric", MinGroupSize: 2, RequireMinFixedPrefix: 2})
	// trailing-only within subset becomes a*
	want := []string{"a*"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGenerateWildcards_InternalNumeric_Deterministic(t *testing.T) {
	hosts := []string{"api02-x-1", "api01-x-1"}
	got1 := GenerateWildcardsWithOptions(hosts, Options{Mode: "internalNumeric", MinGroupSize: 2})
	got2 := GenerateWildcardsWithOptions([]string{"api01-x-1", "api02-x-1"}, Options{Mode: "internalNumeric", MinGroupSize: 2})
	if len(got1) != len(got2) || got1[0] != got2[0] {
		t.Fatalf("nondeterministic: %v vs %v", got1, got2)
	}
}
