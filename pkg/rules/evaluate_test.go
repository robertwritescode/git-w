package rules

import (
	"fmt"
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

func TestEvaluateRuleMatrix(t *testing.T) {
	actions := []struct {
		name string
		kind Action
		flag string
	}{
		{name: "allow", kind: ActionAllow},
		{name: "block", kind: ActionBlock},
		{name: "warn", kind: ActionWarn},
		{name: "require flag", kind: ActionRequireFlag, flag: "--push-wip"},
	}

	criteriaCases := []struct {
		name          string
		branchName    string
		pattern       string
		untracked     *bool
		explicit      *bool
		hasUpstream   bool
		isExplicit    bool
		expectedMatch bool
	}{
		{name: "no criteria", branchName: "main", expectedMatch: true},
		{name: "pattern only", branchName: "feature/login", pattern: "feature/*", expectedMatch: true},
		{name: "untracked only", branchName: "main", untracked: boolPtr(true), hasUpstream: false, expectedMatch: true},
		{name: "explicit only", branchName: "release/2026/q2", explicit: boolPtr(true), isExplicit: true, expectedMatch: true},
		{name: "pattern and untracked", branchName: "feature/login", pattern: "feature/*", untracked: boolPtr(true), hasUpstream: false, expectedMatch: true},
		{name: "pattern and explicit", branchName: "release/2026/q2", pattern: "release/**", explicit: boolPtr(true), isExplicit: true, expectedMatch: true},
		{name: "untracked and explicit", branchName: "main", untracked: boolPtr(false), hasUpstream: true, explicit: boolPtr(true), isExplicit: true, expectedMatch: true},
		{name: "pattern untracked and explicit", branchName: "feature/login", pattern: "feature/*", untracked: boolPtr(true), hasUpstream: false, explicit: boolPtr(true), isExplicit: true, expectedMatch: true},
	}

	for _, criteriaCase := range criteriaCases {
		criteriaCase := criteriaCase
		for _, actionCase := range actions {
			actionCase := actionCase
			t.Run(fmt.Sprintf("%s %s", criteriaCase.name, actionCase.name), func(t *testing.T) {
				branch := BranchInfo{
					Name: criteriaCase.branchName,
					HasUpstreamOn: func(remoteName string) bool {
						return remoteName == "origin" && criteriaCase.hasUpstream
					},
					ExplicitOn: func(remoteName string) bool {
						return remoteName == "origin" && criteriaCase.isExplicit
					},
				}

				rules := []BranchRule{{
					Pattern:   criteriaCase.pattern,
					Action:    actionCase.kind,
					Reason:    criteriaCase.name,
					Flag:      actionCase.flag,
					Untracked: criteriaCase.untracked,
					Explicit:  criteriaCase.explicit,
				}}

				action, matched := EvaluateRule(branch, rules, "origin")

				if !criteriaCase.expectedMatch {
					if action != ActionAllow {
						t.Fatalf("action = %q, want %q", action, ActionAllow)
					}

					if matched != nil {
						t.Fatalf("matched = %#v, want nil", matched)
					}

					return
				}

				if action != actionCase.kind {
					t.Fatalf("action = %q, want %q", action, actionCase.kind)
				}

				if matched == nil {
					t.Fatal("matched = nil, want matching rule")
				}

				if matched != &rules[0] {
					t.Fatalf("matched = %p, want %p", matched, &rules[0])
				}

				if matched.Action != actionCase.kind {
					t.Fatalf("matched.Action = %q, want %q", matched.Action, actionCase.kind)
				}

				if actionCase.kind == ActionRequireFlag && matched.Flag != actionCase.flag {
					t.Fatalf("matched.Flag = %q, want %q", matched.Flag, actionCase.flag)
				}
			})
		}
	}
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
