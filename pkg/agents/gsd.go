package agents

import (
	"os"
	"path/filepath"
)

// GSDFramework implements SpecFramework for the GSD (get-shit-done) agentic framework.
var _ SpecFramework = GSDFramework{}

// GSDFramework is the SpecFramework implementation for GSD.
type GSDFramework struct{}

func (GSDFramework) Name() string { return FrameworkGSD }

func (GSDFramework) PlanningDirExists(rootPath string) bool {
	info, err := os.Stat(filepath.Join(rootPath, ".planning"))
	return err == nil && info.IsDir()
}

// InitInstructions returns GSD initialization guidance for a workstream.
// Full content is generated in Phase 9; this stub satisfies the interface.
func (GSDFramework) InitInstructions(workstreamPath string) string {
	return "Run /gsd:new-project --auto inside " + workstreamPath + " to initialize GSD planning state."
}

func (GSDFramework) ProhibitedActions() []ProhibitedAction {
	return []ProhibitedAction{
		{
			Action:      "do not use /gsd:new-workspace or /gsd:new-project workspace scaffolding",
			Reason:      "git-w creates workstreams; GSD initializes .planning/ inside them",
			Alternative: "git w workstream create",
		},
	}
}

func (GSDFramework) WorkspaceCreationProhibited() bool { return true }
