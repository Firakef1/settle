package service

import "strings"

// normalizeEmail returns the email trimmed of whitespace and lowercased.
// Used by every service method that accepts an email so all lookups and stored values agree.
func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
