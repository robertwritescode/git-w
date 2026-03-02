package workspace_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/robertwritescode/git-w/pkg/workspace"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

type syncPushEnabledCase struct {
	name string
	meta workspace.WorkspaceMeta
	want bool
}

var syncPushEnabledCases = func() []syncPushEnabledCase {
	trueValue := true
	falseValue := false
	return []syncPushEnabledCase{
		{name: "nil defaults true", meta: workspace.WorkspaceMeta{}, want: true},
		{name: "explicit true", meta: workspace.WorkspaceMeta{SyncPush: &trueValue}, want: true},
		{name: "explicit false", meta: workspace.WorkspaceMeta{SyncPush: &falseValue}, want: false},
	}
}()

func TestConfigSuite(t *testing.T) {
	testutil.RunSuite(t, new(ConfigSuite))
}

func (s *ConfigSuite) TestSyncPushEnabled() {
	for _, tt := range syncPushEnabledCases {
		s.Run(tt.name, func() {
			cfg := workspace.WorkspaceConfig{Workspace: tt.meta}
			s.Equal(tt.want, cfg.SyncPushEnabled())
		})
	}
}
