package rules

import (
	"reflect"
	"testing"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestRuleTypes(t *testing.T) {
	if ActionAllow != "allow" {
		t.Fatalf("ActionAllow = %q, want %q", ActionAllow, "allow")
	}

	if ActionBlock != "block" {
		t.Fatalf("ActionBlock = %q, want %q", ActionBlock, "block")
	}

	if ActionWarn != "warn" {
		t.Fatalf("ActionWarn = %q, want %q", ActionWarn, "warn")
	}

	if ActionRequireFlag != "require-flag" {
		t.Fatalf("ActionRequireFlag = %q, want %q", ActionRequireFlag, "require-flag")
	}

	ruleType := reflect.TypeOf(BranchRule{})
	assertRuleField(t, ruleType, "Pattern", reflect.TypeFor[string]())
	assertRuleField(t, ruleType, "Action", reflect.TypeFor[Action]())
	assertRuleField(t, ruleType, "Reason", reflect.TypeFor[string]())
	assertRuleField(t, ruleType, "Flag", reflect.TypeFor[string]())
	assertRuleField(t, ruleType, "Untracked", reflect.TypeFor[*bool]())
	assertRuleField(t, ruleType, "Explicit", reflect.TypeFor[*bool]())
}

func assertRuleField(t *testing.T, ruleType reflect.Type, name string, want reflect.Type) {
	t.Helper()

	field, ok := ruleType.FieldByName(name)
	if !ok {
		t.Fatalf("BranchRule missing field %q", name)
	}

	if field.Type != want {
		t.Fatalf("BranchRule.%s type = %v, want %v", name, field.Type, want)
	}
}

func TestEvaluateRule(t *testing.T) {
	tests := []struct {
		name       string
		branch     BranchInfo
		rules      []BranchRule
		remoteName string
		wantAction Action
		wantMatch  *int
		wantFlag   string
	}{
		{
			name:       "returns default allow when no rules match",
			branch:     BranchInfo{Name: "main"},
			rules:      []BranchRule{{Pattern: "release/**", Action: ActionBlock}},
			remoteName: "origin",
			wantAction: ActionAllow,
		},
		{
			name:   "returns the first matching rule",
			branch: BranchInfo{Name: "feature/login"},
			rules: []BranchRule{
				{Pattern: "feature/*", Action: ActionWarn},
				{Pattern: "feature/*", Action: ActionBlock},
			},
			remoteName: "origin",
			wantAction: ActionWarn,
			wantMatch:  intPtr(0),
		},
		{
			name:       "treats empty pattern as wildcard",
			branch:     BranchInfo{Name: "release/v1"},
			rules:      []BranchRule{{Action: ActionBlock}},
			remoteName: "origin",
			wantAction: ActionBlock,
			wantMatch:  intPtr(0),
		},
		{
			name: "matches untracked true when branch lacks upstream",
			branch: BranchInfo{
				Name: "topic",
				HasUpstreamOn: func(remoteName string) bool {
					return remoteName == "other"
				},
			},
			rules:      []BranchRule{{Action: ActionBlock, Untracked: boolPtr(true)}},
			remoteName: "origin",
			wantAction: ActionBlock,
			wantMatch:  intPtr(0),
		},
		{
			name: "matches untracked false when branch has upstream",
			branch: BranchInfo{
				Name: "topic",
				HasUpstreamOn: func(remoteName string) bool {
					return remoteName == "origin"
				},
			},
			rules:      []BranchRule{{Action: ActionWarn, Untracked: boolPtr(false)}},
			remoteName: "origin",
			wantAction: ActionWarn,
			wantMatch:  intPtr(0),
		},
		{
			name: "matches explicit true when branch is explicit on remote",
			branch: BranchInfo{
				Name: "topic",
				ExplicitOn: func(remoteName string) bool {
					return remoteName == "personal"
				},
			},
			rules:      []BranchRule{{Action: ActionAllow, Explicit: boolPtr(true)}},
			remoteName: "personal",
			wantAction: ActionAllow,
			wantMatch:  intPtr(0),
		},
		{
			name: "matches explicit false when branch is not explicit on remote",
			branch: BranchInfo{
				Name: "topic",
				ExplicitOn: func(remoteName string) bool {
					return remoteName == "other"
				},
			},
			rules:      []BranchRule{{Action: ActionBlock, Explicit: boolPtr(false)}},
			remoteName: "origin",
			wantAction: ActionBlock,
			wantMatch:  intPtr(0),
		},
		{
			name: "round trips require flag action and payload",
			branch: BranchInfo{
				Name: "feature/login",
				HasUpstreamOn: func(remoteName string) bool {
					return false
				},
				ExplicitOn: func(remoteName string) bool {
					return true
				},
			},
			rules: []BranchRule{{
				Pattern:   "feature/*",
				Action:    ActionRequireFlag,
				Flag:      "--push-wip",
				Untracked: boolPtr(true),
				Explicit:  boolPtr(true),
			}},
			remoteName: "personal",
			wantAction: ActionRequireFlag,
			wantMatch:  intPtr(0),
			wantFlag:   "--push-wip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, matched := EvaluateRule(tt.branch, tt.rules, tt.remoteName)

			if action != tt.wantAction {
				t.Fatalf("action = %q, want %q", action, tt.wantAction)
			}

			assertMatchedRule(t, matched, tt.rules, tt.wantMatch)

			if tt.wantFlag != "" && matched.Flag != tt.wantFlag {
				t.Fatalf("matched.Flag = %q, want %q", matched.Flag, tt.wantFlag)
			}
		})
	}

	t.Run("forwards the provided remote name to both predicates", func(t *testing.T) {
		var upstreamRemote string
		var explicitRemote string

		_, matched := EvaluateRule(
			BranchInfo{
				Name: "feature/login",
				HasUpstreamOn: func(remoteName string) bool {
					upstreamRemote = remoteName
					return false
				},
				ExplicitOn: func(remoteName string) bool {
					explicitRemote = remoteName
					return true
				},
			},
			[]BranchRule{{Action: ActionRequireFlag, Flag: "--push-wip", Untracked: boolPtr(true), Explicit: boolPtr(true)}},
			"personal",
		)

		if matched == nil {
			t.Fatal("matched = nil, want matching rule")
		}

		if upstreamRemote != "personal" {
			t.Fatalf("upstream remote = %q, want %q", upstreamRemote, "personal")
		}

		if explicitRemote != "personal" {
			t.Fatalf("explicit remote = %q, want %q", explicitRemote, "personal")
		}
	})
}

func intPtr(v int) *int {
	return &v
}

func assertMatchedRule(t *testing.T, matched *BranchRule, rules []BranchRule, wantMatch *int) {
	t.Helper()

	if wantMatch == nil {
		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}

		return
	}

	if matched == nil {
		t.Fatal("matched = nil, want matching rule")
	}

	if matched != &rules[*wantMatch] {
		t.Fatalf("matched = %p, want %p", matched, &rules[*wantMatch])
	}
}
