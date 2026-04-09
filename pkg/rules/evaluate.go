package rules

// EvaluateRule returns the action and first matching rule for a branch.
// Criteria that cannot be evaluated because a required predicate is missing do not match.
func EvaluateRule(branch BranchInfo, rules []BranchRule, remoteName string) (Action, *BranchRule) {
	for i := range rules {
		if !matchesRule(branch, rules[i], remoteName) {
			continue
		}

		return rules[i].Action, &rules[i]
	}

	return ActionAllow, nil
}

func matchesRule(branch BranchInfo, rule BranchRule, remoteName string) bool {
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

func matchesUntracked(branch BranchInfo, want *bool, remoteName string) bool {
	if want == nil || branch.HasUpstreamOn == nil {
		return want == nil
	}

	hasUpstream := branch.HasUpstreamOn(remoteName)
	if *want {
		return !hasUpstream
	}

	return hasUpstream
}

func matchesExplicit(branch BranchInfo, want *bool, remoteName string) bool {
	if want == nil || branch.ExplicitOn == nil {
		return want == nil
	}

	isExplicit := branch.ExplicitOn(remoteName)
	if *want {
		return isExplicit
	}

	return !isExplicit
}
