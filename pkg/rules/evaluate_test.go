package rules

import (
	"reflect"
	"testing"
)

func boolPtr(v bool) *bool {
	return &v
}

func intPtr(v int) *int {
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

	ruleType := reflect.TypeOf(Rule{})
	assertRuleField(t, ruleType, "Pattern", reflect.TypeFor[string]())
	assertRuleField(t, ruleType, "Action", reflect.TypeFor[Action]())
	assertRuleField(t, ruleType, "Reason", reflect.TypeFor[string]())
	assertRuleField(t, ruleType, "Flag", reflect.TypeFor[string]())
	assertRuleField(t, ruleType, "Untracked", reflect.TypeFor[*bool]())
	assertRuleField(t, ruleType, "Explicit", reflect.TypeFor[*bool]())
}

func TestEvaluate_returnsFirstMatchingRule(t *testing.T) {
	rules := []Rule{
		newRule(ActionWarn),
		newRule(ActionBlock),
	}

	action, matched := Evaluate(newBranch("feature/login"), rules, "origin")

	if action != ActionWarn {
		t.Fatalf("action = %q, want %q", action, ActionWarn)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_returnsDefaultAllowWhenNoRuleMatches(t *testing.T) {
	action, matched := Evaluate(
		newBranch("main"),
		[]Rule{newRule(ActionBlock, withPattern("feature/*"))},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluate_matchesCatchAllRule(t *testing.T) {
	rules := []Rule{newRule(ActionBlock)}

	action, matched := Evaluate(newBranch("release/2026/q2"), rules, "origin")

	if action != ActionBlock {
		t.Fatalf("action = %q, want %q", action, ActionBlock)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesPatternRule(t *testing.T) {
	rules := []Rule{newRule(ActionBlock, withPattern("feature/*"))}

	action, matched := Evaluate(newBranch("feature/login"), rules, "origin")

	if action != ActionBlock {
		t.Fatalf("action = %q, want %q", action, ActionBlock)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesUntrackedRule(t *testing.T) {
	rules := []Rule{newRule(ActionWarn, withUntracked(true))}

	action, matched := Evaluate(newBranch("main", withUpstream(false)), rules, "origin")

	if action != ActionWarn {
		t.Fatalf("action = %q, want %q", action, ActionWarn)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesTrackedRule(t *testing.T) {
	rules := []Rule{newRule(ActionWarn, withUntracked(false))}

	action, matched := Evaluate(newBranch("main", withUpstream(true)), rules, "origin")

	if action != ActionWarn {
		t.Fatalf("action = %q, want %q", action, ActionWarn)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesExplicitRule(t *testing.T) {
	rules := []Rule{newRule(ActionBlock, withExplicit(true))}

	action, matched := Evaluate(newBranch("release/2026/q2", withExplicitBranch(true)), rules, "origin")

	if action != ActionBlock {
		t.Fatalf("action = %q, want %q", action, ActionBlock)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesImplicitRule(t *testing.T) {
	rules := []Rule{newRule(ActionBlock, withExplicit(false))}

	action, matched := Evaluate(newBranch("release/2026/q2", withExplicitBranch(false)), rules, "origin")

	if action != ActionBlock {
		t.Fatalf("action = %q, want %q", action, ActionBlock)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesPatternAndUntrackedRule(t *testing.T) {
	rules := []Rule{newRule(ActionBlock, withPattern("feature/*"), withUntracked(true))}

	action, matched := Evaluate(newBranch("feature/login", withUpstream(false)), rules, "origin")

	if action != ActionBlock {
		t.Fatalf("action = %q, want %q", action, ActionBlock)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesPatternAndExplicitRule(t *testing.T) {
	rules := []Rule{newRule(ActionWarn, withPattern("release/**"), withExplicit(false))}

	action, matched := Evaluate(newBranch("release/2026/q2", withExplicitBranch(false)), rules, "origin")

	if action != ActionWarn {
		t.Fatalf("action = %q, want %q", action, ActionWarn)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesUntrackedAndExplicitRule(t *testing.T) {
	rules := []Rule{newRule(ActionWarn, withUntracked(false), withExplicit(true))}

	action, matched := Evaluate(
		newBranch("main", withUpstream(true), withExplicitBranch(true)),
		rules,
		"origin",
	)

	if action != ActionWarn {
		t.Fatalf("action = %q, want %q", action, ActionWarn)
	}

	assertMatchedRule(t, matched, rules, intPtr(0))
}

func TestEvaluate_matchesPatternUntrackedAndExplicitRule(t *testing.T) {
	rules := []Rule{newRule(ActionRequireFlag, withPattern("feature/*"), withUntracked(true), withExplicit(true), withFlag("--push-wip"))}

	action, matched := Evaluate(
		newBranch("feature/login", withUpstream(false), withExplicitBranch(true)),
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

func TestEvaluate_emptyPatternActsAsNoBranchFilter(t *testing.T) {
	rules := []Rule{newRule(ActionRequireFlag, withFlag("--push-wip"), withExplicit(true))}

	action, matched := Evaluate(
		newBranch("arbitrary/topic/name", withExplicitBranch(true)),
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

func TestEvaluate_returnsDefaultAllowWhenExplicitCriterionFails(t *testing.T) {
	action, matched := Evaluate(
		newBranch("feature/login", withExplicitBranch(false)),
		[]Rule{newRule(ActionBlock, withPattern("feature/*"), withExplicit(true))},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluate_returnsDefaultAllowWhenPatternFails(t *testing.T) {
	action, matched := Evaluate(
		newBranch("feature/login", withUpstream(false)),
		[]Rule{newRule(ActionWarn, withPattern("main"), withUntracked(true))},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluate_returnsDefaultAllowWhenImplicitCriterionFails(t *testing.T) {
	action, matched := Evaluate(
		newBranch("release/2026/q2", withExplicitBranch(true)),
		[]Rule{newRule(ActionBlock, withPattern("release/**"), withExplicit(false))},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluate_doesNotMatchWhenUpstreamPredicateIsMissing(t *testing.T) {
	action, matched := Evaluate(
		newBranch("feature/login"),
		[]Rule{newRule(ActionBlock, withPattern("feature/*"), withUntracked(true))},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluate_doesNotMatchWhenExplicitPredicateIsMissing(t *testing.T) {
	action, matched := Evaluate(
		newBranch("feature/login"),
		[]Rule{newRule(ActionWarn, withPattern("feature/*"), withExplicit(true))},
		"origin",
	)

	if action != ActionAllow {
		t.Fatalf("action = %q, want %q", action, ActionAllow)
	}

	if matched != nil {
		t.Fatalf("matched = %#v, want nil", matched)
	}
}

func TestEvaluate_forwardsProvidedRemoteName(t *testing.T) {
	var upstreamRemote string
	var explicitRemote string

	rules := []Rule{newRule(ActionRequireFlag, withFlag("--push-wip"), withUntracked(true), withExplicit(true))}

	_, matched := Evaluate(
		newBranch(
			"feature/login",
			withUpstreamForRemote(&upstreamRemote, false),
			withExplicitForRemote(&explicitRemote, true),
		),
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

func newBranch(name string, opts ...func(*Branch)) Branch {
	branch := Branch{Name: name}
	for _, opt := range opts {
		opt(&branch)
	}

	return branch
}

func withUpstream(hasUpstream bool) func(*Branch) {
	return func(branch *Branch) {
		branch.HasUpstreamOn = func(remoteName string) bool {
			return hasUpstream
		}
	}
}

func withExplicitBranch(isExplicit bool) func(*Branch) {
	return func(branch *Branch) {
		branch.ExplicitOn = func(remoteName string) bool {
			return isExplicit
		}
	}
}

func withUpstreamForRemote(remoteSeen *string, hasUpstream bool) func(*Branch) {
	return func(branch *Branch) {
		branch.HasUpstreamOn = func(remoteName string) bool {
			*remoteSeen = remoteName
			return hasUpstream
		}
	}
}

func withExplicitForRemote(remoteSeen *string, isExplicit bool) func(*Branch) {
	return func(branch *Branch) {
		branch.ExplicitOn = func(remoteName string) bool {
			*remoteSeen = remoteName
			return isExplicit
		}
	}
}

func newRule(action Action, opts ...func(*Rule)) Rule {
	rule := Rule{Action: action}
	for _, opt := range opts {
		opt(&rule)
	}

	return rule
}

func withPattern(pattern string) func(*Rule) {
	return func(rule *Rule) {
		rule.Pattern = pattern
	}
}

func withFlag(flag string) func(*Rule) {
	return func(rule *Rule) {
		rule.Flag = flag
	}
}

func withUntracked(want bool) func(*Rule) {
	return func(rule *Rule) {
		rule.Untracked = boolPtr(want)
	}
}

func withExplicit(want bool) func(*Rule) {
	return func(rule *Rule) {
		rule.Explicit = boolPtr(want)
	}
}

func assertRuleField(t *testing.T, ruleType reflect.Type, name string, want reflect.Type) {
	t.Helper()

	field, ok := ruleType.FieldByName(name)
	if !ok {
		t.Fatalf("Rule missing field %q", name)
	}

	if field.Type != want {
		t.Fatalf("Rule.%s type = %v, want %v", name, field.Type, want)
	}
}

func assertMatchedRule(t *testing.T, matched *Rule, rules []Rule, wantMatch *int) {
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
