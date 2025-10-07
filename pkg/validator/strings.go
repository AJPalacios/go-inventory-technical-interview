package validator

import "strings"

// Contains checks if a string slice contains a specific item (case-insensitive).
//
// This utility function provides a reusable way to check slice membership
// commonly needed in validation scenarios.
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// ContainsCaseSensitive checks if a string slice contains a specific item (case-sensitive).
func ContainsCaseSensitive(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
