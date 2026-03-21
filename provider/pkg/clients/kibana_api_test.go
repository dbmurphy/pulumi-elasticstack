package clients

import "testing"

func TestSpacePath(t *testing.T) {
	tests := []struct {
		name    string
		spaceID string
		path    string
		want    string
	}{
		{
			name:    "empty spaceID returns path as-is",
			spaceID: "",
			path:    "/api/saved_objects",
			want:    "/api/saved_objects",
		},
		{
			name:    "default spaceID returns path as-is",
			spaceID: "default",
			path:    "/api/saved_objects",
			want:    "/api/saved_objects",
		},
		{
			name:    "custom spaceID prefixes path with leading slash",
			spaceID: "my-space",
			path:    "/api/saved_objects",
			want:    "/s/my-space/api/saved_objects",
		},
		{
			name:    "custom spaceID with path without leading slash",
			spaceID: "my-space",
			path:    "api/saved_objects",
			want:    "/s/my-space/api/saved_objects",
		},
		{
			name:    "empty path with empty spaceID",
			spaceID: "",
			path:    "",
			want:    "",
		},
		{
			name:    "empty path with custom spaceID",
			spaceID: "my-space",
			path:    "",
			want:    "/s/my-space/",
		},
		{
			name:    "path with multiple leading slashes trimmed",
			spaceID: "my-space",
			path:    "///api/test",
			want:    "/s/my-space/api/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SpacePath(tt.spaceID, tt.path)
			if got != tt.want {
				t.Errorf("SpacePath(%q, %q) = %q, want %q", tt.spaceID, tt.path, got, tt.want)
			}
		})
	}
}
