package rules

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		branch  string
		want    bool
	}{
		{name: "literal match", pattern: "main", branch: "main", want: true},
		{name: "single star stays within segment", pattern: "feature/*", branch: "feature/login", want: true},
		{name: "double star crosses segments", pattern: "release/**", branch: "release/v1/hotfix", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.branch); got != tt.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.branch, got, tt.want)
			}
		})
	}
}
