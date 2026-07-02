package enums

import "testing"

func TestStatusForVerdict(t *testing.T) {
	cases := map[string]string{
		VerdictApprove:        ChangesetApproved,
		VerdictRequestChanges: ChangesetChangesRequested,
		VerdictDeny:           ChangesetDenied,
		"":                    "",
		"bogus":               "",
	}
	for verdict, want := range cases {
		if got := StatusForVerdict(verdict); got != want {
			t.Errorf("StatusForVerdict(%q) = %q, want %q", verdict, got, want)
		}
	}
}

func TestReviewVerdictSet(t *testing.T) {
	for _, v := range []string{"approve", "deny", "request_changes"} {
		if !Valid(ReviewVerdict, v) {
			t.Errorf("verdict %q should be valid", v)
		}
	}
	for _, v := range []string{"maybe", "approved", "reject", ""} {
		if v != "" && Valid(ReviewVerdict, v) {
			t.Errorf("verdict %q should be invalid", v)
		}
	}
	// Every allowed verdict must map to a changeset status (keeps the two in lockstep).
	for _, v := range ReviewVerdict {
		if StatusForVerdict(v) == "" {
			t.Errorf("verdict %q has no changeset status mapping", v)
		}
	}
}

func TestCommentSubjectTypeSet(t *testing.T) {
	for _, st := range []string{"requirement", "spec", "user_story", "test_case", "entity", "deliverable"} {
		if !Valid(CommentSubjectType, st) {
			t.Errorf("subject type %q should be valid", st)
		}
	}
	for _, st := range []string{"domain", "milestone", "glossary_term", "capability"} {
		if Valid(CommentSubjectType, st) {
			t.Errorf("subject type %q should NOT be a comment subject", st)
		}
	}
}
