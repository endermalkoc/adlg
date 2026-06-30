package ids

import "testing"

func TestNew_uniqueAndULIDShaped(t *testing.T) {
	a, b := New(), New()
	if a == b {
		t.Fatal("two New() calls collided")
	}
	if len(a) != 26 { // canonical ULID is 26 Crockford-base32 chars
		t.Fatalf("ULID length = %d, want 26 (%q)", len(a), a)
	}
}

func TestRel_deterministicAndIdentitySensitive(t *testing.T) {
	a := Rel("requirement", "r2", "requirement", "r1", "refines")
	b := Rel("requirement", "r2", "requirement", "r1", "refines")
	if a != b {
		t.Fatalf("Rel not deterministic: %s != %s", a, b)
	}
	// A different identity (different kind) must produce a different id.
	if c := Rel("requirement", "r2", "requirement", "r1", "relates"); a == c {
		t.Fatal("distinct identities produced the same id")
	}
	// Component boundaries must matter (no separator ambiguity).
	if Rel("ab", "c") == Rel("a", "bc") {
		t.Fatal("separator ambiguity: (ab,c) collided with (a,bc)")
	}
}
