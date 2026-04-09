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
		{name: "literal mismatch across slash", pattern: "main", branch: "feature/main", want: false},
		{name: "single star stays within segment", pattern: "feature/*", branch: "feature/login", want: true},
		{name: "single star does not cross slash", pattern: "feature/*", branch: "feature/ui/login", want: false},
		{name: "double star matches one segment", pattern: "release/**", branch: "release/v1", want: true},
		{name: "double star crosses segments", pattern: "release/**", branch: "release/v1/hotfix", want: true},
		{name: "double star matches main", pattern: "**", branch: "main", want: true},
		{name: "double star matches nested branch", pattern: "**", branch: "feature/ui/login", want: true},
		{name: "single star matches middle segment", pattern: "hotfix/*/ready", branch: "hotfix/api/ready", want: true},
		{name: "single star rejects extra segment", pattern: "hotfix/*/ready", branch: "hotfix/api/v2/ready", want: false},
		{name: "empty pattern matches empty branch", pattern: "", branch: "", want: true},
		{name: "empty pattern rejects non-empty branch", pattern: "", branch: "main", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.branch); got != tt.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", tt.pattern, tt.branch, got, tt.want)
			}
		})
	}
}
