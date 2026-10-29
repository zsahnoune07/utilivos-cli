package publish

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"  Foo Bar  ", "foo-bar"},
		{"developer_tools", "developer-tools"},
		{"", ""},
		{"a", "a"},
	}
	for _, tc := range tests {
		if got := Slugify(tc.in); got != tc.want {
			t.Fatalf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
