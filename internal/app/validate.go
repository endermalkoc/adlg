package app

import (
	"fmt"

	"github.com/endermalkoc/asdf/internal/enums"
)

// ValidateEnum checks that value is in the allowed set. An empty value passes
// (optional fields); callers enforce required-ness separately with ValidateRequired.
func ValidateEnum(field, value string, allowed []string) error {
	if value == "" {
		return nil
	}
	if !enums.Valid(allowed, value) {
		return fmt.Errorf("invalid %s %q (allowed: %v)", field, value, allowed)
	}
	return nil
}

// ValidateRequired checks that value is non-empty.
func ValidateRequired(field, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}
