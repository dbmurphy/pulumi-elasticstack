package cluster

import "testing"

func TestJsonStringsEqual(t *testing.T) {
	tests := []struct {
		name string
		a    *string
		b    *string
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil b non-nil",
			a:    nil,
			b:    ptrOf(`{"key":"value"}`),
			want: false,
		},
		{
			name: "a non-nil b nil",
			a:    ptrOf(`{"key":"value"}`),
			b:    nil,
			want: false,
		},
		{
			name: "equal JSON same formatting",
			a:    ptrOf(`{"key":"value"}`),
			b:    ptrOf(`{"key":"value"}`),
			want: true,
		},
		{
			name: "equal JSON different formatting",
			a:    ptrOf(`{"key": "value"}`),
			b:    ptrOf(`{"key":"value"}`),
			want: true,
		},
		{
			name: "equal JSON different key order",
			a:    ptrOf(`{"a":1,"b":2}`),
			b:    ptrOf(`{"b":2,"a":1}`),
			want: true,
		},
		{
			name: "different values",
			a:    ptrOf(`{"key":"value1"}`),
			b:    ptrOf(`{"key":"value2"}`),
			want: false,
		},
		{
			name: "invalid JSON falls back to string comparison - equal",
			a:    ptrOf("not json"),
			b:    ptrOf("not json"),
			want: true,
		},
		{
			name: "invalid JSON falls back to string comparison - different",
			a:    ptrOf("not json"),
			b:    ptrOf("also not json"),
			want: false,
		},
		{
			name: "a valid b invalid falls back to string comparison",
			a:    ptrOf(`{"key":"value"}`),
			b:    ptrOf("not json"),
			want: false,
		},
		{
			name: "empty JSON objects",
			a:    ptrOf(`{}`),
			b:    ptrOf(`{}`),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonStringsEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("jsonStringsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPtrOf(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "non-empty string", input: "hello"},
		{name: "empty string", input: ""},
		{name: "json string", input: `{"key":"value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ptrOf(tt.input)
			if got == nil {
				t.Fatal("ptrOf returned nil")
			}
			if *got != tt.input {
				t.Errorf("ptrOf(%q) = %q, want %q", tt.input, *got, tt.input)
			}
		})
	}
}
