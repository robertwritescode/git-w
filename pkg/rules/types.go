package rules

// BranchInfo is the branch snapshot consumed by the branch rule engine.
type BranchInfo struct {
	Name          string
	HasUpstreamOn func(remoteName string) bool
	ExplicitOn    func(remoteName string) bool
}
