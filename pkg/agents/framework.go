package agents

// SpecFramework describes the contract that any agentic spec framework must
// satisfy to integrate with git-w's agent context layer.
//
// At v2 launch only GSD is supported. Future frameworks satisfy this interface
// without requiring changes to any other package.
type SpecFramework interface {
	// Name returns the canonical identifier for this framework (e.g. "gsd").
	Name() string

	// PlanningDirExists reports whether the framework's planning state
	// directory is present at the given root path.
	PlanningDirExists(rootPath string) bool

	// InitInstructions returns the agent-readable string explaining how to
	// initialize this framework's planning state in a new workstream directory.
	InitInstructions(workstreamPath string) string

	// ProhibitedActions returns actions the framework must not perform inside
	// a git-w-managed environment, as (action, alternative) pairs.
	ProhibitedActions() []ProhibitedAction

	// WorkspaceCreationProhibited reports whether this framework's own
	// workspace-creation command must be suppressed inside a git-w workstream.
	WorkspaceCreationProhibited() bool
}

// ProhibitedAction is a (what, why, alternative) triple surfaced in AGENTS.md.
type ProhibitedAction struct {
	Action      string
	Reason      string
	Alternative string
}
