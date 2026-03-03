package config_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

type syncPushEnabledCase struct {
	name string
	meta config.WorkspaceMeta
	want bool
}

var syncPushEnabledCases = func() []syncPushEnabledCase {
	trueValue := true
	falseValue := false
	return []syncPushEnabledCase{
		{name: "nil defaults true", meta: config.WorkspaceMeta{}, want: true},
		{name: "explicit true", meta: config.WorkspaceMeta{SyncPush: &trueValue}, want: true},
		{name: "explicit false", meta: config.WorkspaceMeta{SyncPush: &falseValue}, want: false},
	}
}()

func TestConfigSuite(t *testing.T) {
	testutil.RunSuite(t, new(ConfigSuite))
}

func (s *ConfigSuite) TestSyncPushEnabled() {
	for _, tt := range syncPushEnabledCases {
		s.Run(tt.name, func() {
			cfg := config.WorkspaceConfig{Workspace: tt.meta}
			s.Equal(tt.want, cfg.SyncPushEnabled())
		})
	}
}
