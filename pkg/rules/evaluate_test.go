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
		{name: "pattern and explicit", branchName: "release/2026/q2", pattern: "release/**", explicit: boolPtr(false), isExplicit: false, expectedMatch: true},
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

func TestEvaluateRuleFirstMatchWins(t *testing.T) {
	rules := []BranchRule{
		{Action: ActionWarn},
		{Action: ActionBlock},
	}

	action, matched := EvaluateRule(BranchInfo{Name: "feature/login"}, rules, "origin")

	if action != ActionWarn {
		t.Fatalf("action = %q, want %q", action, ActionWarn)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluateRuleDefaultAllowWithNilRule(t *testing.T) {
	action, matched := EvaluateRule(
		BranchInfo{Name: "main"},
		[]BranchRule{{Pattern: "feature/*", Action: ActionBlock}},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluateRuleNoCriteriaCatchAll(t *testing.T) {
	rules := []BranchRule{{Action: ActionBlock}}

	action, matched := EvaluateRule(BranchInfo{Name: "release/2026/q2"}, rules, "origin")

	if action != ActionBlock {
		t.Fatalf("action = %q, want %q", action, ActionBlock)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluateRuleEmptyPatternActsLikeNoBranchFilter(t *testing.T) {
	rules := []BranchRule{{
		Pattern:  "",
		Action:   ActionRequireFlag,
		Flag:     "--push-wip",
		Explicit: boolPtr(true),
	}}

	action, matched := EvaluateRule(
		BranchInfo{
			Name: "arbitrary/topic/name",
			ExplicitOn: func(remoteName string) bool {
				return remoteName == "origin"
			},
		},
		rules,
		"origin",
	)

	if action != ActionRequireFlag {
		t.Fatalf("action = %q, want %q", action, ActionRequireFlag)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))

	if matched.Flag != "--push-wip" {
		t.Fatalf("matched.Flag = %q, want %q", matched.Flag, "--push-wip")
	}
}

func TestEvaluateRuleConflictingCriteriaReturnsDefaultAllow(t *testing.T) {
	t.Run("pattern matches but explicit fails", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{
				Name: "feature/login",
				ExplicitOn: func(remoteName string) bool {
					return false
				},
			},
			[]BranchRule{{
				Pattern:  "feature/*",
				Action:   ActionBlock,
				Explicit: boolPtr(true),
			}},
			"origin",
		)

		if action != ActionAllow {
			t.Fatalf("action = %q, want %q", action, ActionAllow)
		}

		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}
	})

	t.Run("untracked matches but pattern fails", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{
				Name: "feature/login",
				HasUpstreamOn: func(remoteName string) bool {
					return false
				},
			},
			[]BranchRule{{
				Pattern:   "main",
				Action:    ActionWarn,
				Untracked: boolPtr(true),
			}},
			"origin",
		)

		if action != ActionAllow {
			t.Fatalf("action = %q, want %q", action, ActionAllow)
		}

		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}
	})

	t.Run("pattern matches but explicit false fails on explicit branch", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{
				Name: "release/2026/q2",
				ExplicitOn: func(remoteName string) bool {
					return remoteName == "origin"
				},
			},
			[]BranchRule{{
				Pattern:  "release/**",
				Action:   ActionBlock,
				Explicit: boolPtr(false),
			}},
			"origin",
		)

		if action != ActionAllow {
			t.Fatalf("action = %q, want %q", action, ActionAllow)
		}

		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}
	})
}

func TestEvaluateRuleMissingPredicatesDoNotMatch(t *testing.T) {
	t.Run("untracked criterion with nil upstream predicate", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{Name: "feature/login"},
			[]BranchRule{{
				Pattern:   "feature/*",
				Action:    ActionBlock,
				Untracked: boolPtr(true),
			}},
			"origin",
		)

		if action != ActionAllow {
			t.Fatalf("action = %q, want %q", action, ActionAllow)
		}

		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}
	})

	t.Run("explicit criterion with nil explicit predicate", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{Name: "feature/login"},
			[]BranchRule{{
				Pattern:  "feature/*",
				Action:   ActionWarn,
				Explicit: boolPtr(true),
			}},
			"origin",
		)

		if action != ActionAllow {
			t.Fatalf("action = %q, want %q", action, ActionAllow)
		}

		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}
	})
}

func TestEvaluateRuleForwardsProvidedRemoteName(t *testing.T) {
	var upstreamRemote string
	var explicitRemote string

	rules := []BranchRule{{
		Action:    ActionRequireFlag,
		Flag:      "--push-wip",
		Untracked: boolPtr(true),
		Explicit:  boolPtr(true),
	}}

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
		rules,
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
