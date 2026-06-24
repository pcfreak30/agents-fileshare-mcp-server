package share

import (
	"testing"
)

func TestGenerateID_Length(t *testing.T) {
	for _, length := range []int{1, 4, 8, 16, 32} {
		id := GenerateID(length)
		if len(id) != length {
			t.Errorf("GenerateID(%d) length = %d, want %d", length, len(id), length)
		}
	}
}

func TestGenerateID_Base62(t *testing.T) {
	const base62 = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	for i := 0; i < 100; i++ {
		id := GenerateID(8)
		for _, c := range id {
			if !contains(base62, c) {
				t.Errorf("GenerateID produced non-base62 char %q in %q", c, id)
			}
		}
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateID(8)
		if seen[id] {
			t.Errorf("duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}

func contains(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
