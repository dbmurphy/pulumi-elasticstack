package dashboard

import (
	"testing"
)

func strPtr(s string) *string { return &s }

func TestDerefString(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want string
	}{
		{"nil returns empty", nil, ""},
		{"non-nil returns value", strPtr("my-space"), "my-space"},
		{"empty string returns empty", strPtr(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derefString(tt.in)
			if got != tt.want {
				t.Errorf("derefString() = %q, want %q", got, tt.want)
			}
		})
	}
}
