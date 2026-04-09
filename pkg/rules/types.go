package rules

// Branch is the branch snapshot consumed by the branch rule engine.
type Branch struct {
	Name          string
	HasUpstreamOn func(remoteName string) bool
	ExplicitOn    func(remoteName string) bool
}
