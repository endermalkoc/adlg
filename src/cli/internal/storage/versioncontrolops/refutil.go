package versioncontrolops

import (
	"fmt"
	"regexp"
	"strings"
)

// This file holds two small, domain-agnostic helpers that beads kept in its
// issueops package. They were relocated here during the salvage so
// versioncontrolops carries no dependency on the issue domain.

// validRefPattern matches valid Dolt commit hashes (32 hex chars) or branch
// names. Allows dots and slashes for branch names like "release/v2.0" or
// "feature/auth.flow".
var validRefPattern = regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)

// ValidateRef checks if a ref string is safe to use in AS OF queries.
// Refs must be non-empty, <= 128 chars, and match [a-zA-Z0-9_./-]+.
func ValidateRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("ref cannot be empty")
	}
	if len(ref) > 128 {
		return fmt.Errorf("ref too long")
	}
	if !validRefPattern.MatchString(ref) {
		return fmt.Errorf("invalid ref format: %s", ref)
	}
	return nil
}

// IsNothingToCommitError returns true if the error indicates there was nothing
// to commit (Dolt may report this even when dolt_status showed changes).
func IsNothingToCommitError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "nothing to commit") {
		return true
	}
	if strings.Contains(s, "no changes") && strings.Contains(s, "commit") {
		return true
	}
	return false
}
