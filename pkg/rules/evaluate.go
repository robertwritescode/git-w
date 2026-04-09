package rules

// Evaluate returns the action and first matching rule for a branch.
// Criteria that cannot be evaluated because a required predicate is missing do not match.
func Evaluate(branch Branch, rules []Rule, remoteName string) (Action, *Rule) {
	for i := range rules {
		if !matchesRule(branch, rules[i], remoteName) {
			continue
		}

		return rules[i].Action, &rules[i]
	}

	return ActionAllow, nil
}

func matchesRule(branch Branch, rule Rule, remoteName string) bool {
	if !matchesPattern(rule.Pattern, branch.Name) {
		return false
	}

	if !matchesUntracked(branch, rule.Untracked, remoteName) {
		return false
	}

	if !matchesExplicit(branch, rule.Explicit, remoteName) {
		return false
	}

	return true
}

func matchesPattern(pattern, branchName string) bool {
	if pattern == "" {
		return true
	}

	return Match(pattern, branchName)
}

func matchesUntracked(branch Branch, want *bool, remoteName string) bool {
	if want == nil || branch.HasUpstreamOn == nil {
		return want == nil
	}

	hasUpstream := branch.HasUpstreamOn(remoteName)
	if *want {
		return !hasUpstream
	}

	return hasUpstream
}

func matchesExplicit(branch Branch, want *bool, remoteName string) bool {
	if want == nil || branch.ExplicitOn == nil {
		return want == nil
	}

	isExplicit := branch.ExplicitOn(remoteName)
	if *want {
		return isExplicit
	}

	return !isExplicit
}
