package rules

// Action is the evaluator outcome for a branch rule.
type Action string

const (
	ActionAllow       Action = "allow"
	ActionBlock       Action = "block"
	ActionWarn        Action = "warn"
	ActionRequireFlag Action = "require-flag"
)

// BranchRule is the engine-local branch rule shape used by the evaluator.
type BranchRule struct {
	Pattern   string
	Action    Action
	Reason    string
	Flag      string
	Untracked *bool
	Explicit  *bool
}
