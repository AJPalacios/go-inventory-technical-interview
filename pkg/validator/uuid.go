// Package validator provides common validation utilities.
//
// This package contains reusable validation functions that can be used
// across different layers of the application without creating dependencies.
package validator

import "regexp"

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsValidUUID validates if a string is a valid UUID v4 format.
//
// This function uses a pre-compiled regex for performance and validates
// against the standard UUID format: 8-4-4-4-12 hexadecimal digits.
func IsValidUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}
