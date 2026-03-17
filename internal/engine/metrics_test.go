package engine

import "testing"

func TestStatusClass(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{in: 200, want: "2xx"},
		{in: 404, want: "4xx"},
		{in: 503, want: "5xx"},
		{in: 99, want: "unknown"},
		{in: 1000, want: "unknown"},
	}

	for _, tt := range tests {
		got := statusClass(tt.in)
		if got != tt.want {
			t.Fatalf("statusClass(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
