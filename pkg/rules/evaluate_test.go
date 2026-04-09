package rules

import (
	"reflect"
	"testing"
)

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
	t.Run("returns default allow when no rules match", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{Name: "main"},
			[]BranchRule{{Pattern: "release/**", Action: ActionBlock}},
			"origin",
		)

		if action != ActionAllow {
			t.Fatalf("action = %q, want %q", action, ActionAllow)
		}

		if matched != nil {
			t.Fatalf("matched = %#v, want nil", matched)
		}
	})

	t.Run("returns the first matching rule", func(t *testing.T) {
		rules := []BranchRule{
			{Pattern: "feature/*", Action: ActionWarn},
			{Pattern: "feature/*", Action: ActionBlock},
		}

		action, matched := EvaluateRule(BranchInfo{Name: "feature/login"}, rules, "origin")

		if action != ActionWarn {
			t.Fatalf("action = %q, want %q", action, ActionWarn)
		}

		if matched != &rules[0] {
			t.Fatalf("matched = %p, want %p", matched, &rules[0])
		}
	})

	t.Run("treats empty pattern as wildcard", func(t *testing.T) {
		action, matched := EvaluateRule(
			BranchInfo{Name: "release/v1"},
			[]BranchRule{{Action: ActionBlock}},
			"origin",
		)

		if action != ActionBlock {
			t.Fatalf("action = %q, want %q", action, ActionBlock)
		}

		if matched == nil {
			t.Fatal("matched = nil, want matching rule")
		}
	})

	t.Run("uses the provided remote name for predicates", func(t *testing.T) {
		var upstreamRemote string
		var explicitRemote string
		wantTrue := true

		action, matched := EvaluateRule(
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
			[]BranchRule{{Action: ActionRequireFlag, Flag: "--push-wip", Untracked: &wantTrue, Explicit: &wantTrue}},
			"personal",
		)

		if action != ActionRequireFlag {
			t.Fatalf("action = %q, want %q", action, ActionRequireFlag)
		}

		if matched == nil {
			t.Fatal("matched = nil, want matching rule")
		}

		if matched.Flag != "--push-wip" {
			t.Fatalf("matched.Flag = %q, want %q", matched.Flag, "--push-wip")
		}

		if upstreamRemote != "personal" {
			t.Fatalf("upstream remote = %q, want %q", upstreamRemote, "personal")
		}

		if explicitRemote != "personal" {
			t.Fatalf("explicit remote = %q, want %q", explicitRemote, "personal")
		}
	})
}
