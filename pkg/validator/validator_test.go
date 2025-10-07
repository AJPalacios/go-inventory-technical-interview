package validator

import "testing"

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name  string
		uuid  string
		valid bool
	}{
		{
			name:  "valid UUID v4",
			uuid:  "550e8400-e29b-41d4-a716-446655440000",
			valid: true,
		},
		{
			name:  "valid UUID lowercase",
			uuid:  "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
			valid: true,
		},
		{
			name:  "valid UUID uppercase",
			uuid:  "6BA7B810-9DAD-11D1-80B4-00C04FD430C8",
			valid: true,
		},
		{
			name:  "invalid UUID - too short",
			uuid:  "550e8400-e29b-41d4-a716",
			valid: false,
		},
		{
			name:  "invalid UUID - wrong format",
			uuid:  "550e8400e29b41d4a716446655440000",
			valid: false,
		},
		{
			name:  "invalid UUID - invalid characters",
			uuid:  "550e8400-e29b-41d4-a716-44665544000g",
			valid: false,
		},
		{
			name:  "empty string",
			uuid:  "",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidUUID(tt.uuid)
			if result != tt.valid {
				t.Errorf("IsValidUUID(%q) = %v, want %v", tt.uuid, result, tt.valid)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists - exact case",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item exists - different case",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "BANANA",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "orange",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Contains(%v, %q) = %v, want %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

func TestContainsCaseSensitive(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists - exact case",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item exists - different case",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "BANANA",
			expected: false,
		},
		{
			name:     "item does not exist",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "orange",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsCaseSensitive(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("ContainsCaseSensitive(%v, %q) = %v, want %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}
