package rules

// Match reports whether pattern matches branchName using branch-name glob semantics.
func Match(pattern, branchName string) bool {
	return matchRunes([]rune(pattern), []rune(branchName))
}

func matchRunes(pattern, branch []rune) bool {
	if len(pattern) == 0 {
		return len(branch) == 0
	}

	if hasDoubleStar(pattern) {
		return matchDoubleStar(pattern[2:], branch)
	}

	if pattern[0] == '*' {
		return matchSingleStar(pattern[1:], branch)
	}

	if len(branch) == 0 || pattern[0] != branch[0] {
		return false
	}

	return matchRunes(pattern[1:], branch[1:])
}

func hasDoubleStar(pattern []rune) bool {
	return len(pattern) > 1 && pattern[0] == '*' && pattern[1] == '*'
}

func matchDoubleStar(pattern, branch []rune) bool {
	if matchRunes(pattern, branch) {
		return true
	}

	if len(branch) == 0 {
		return false
	}

	return matchDoubleStar(pattern, branch[1:])
}

func matchSingleStar(pattern, branch []rune) bool {
	if matchRunes(pattern, branch) {
		return true
	}

	if len(branch) == 0 || branch[0] == '/' {
		return false
	}

	return matchSingleStar(pattern, branch[1:])
}
