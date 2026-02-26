package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertwritescode/git-workspace/internal/config"
	"github.com/stretchr/testify/suite"
)

type ContextSuite struct {
	suite.Suite
}

func TestContext(t *testing.T) {
	suite.Run(t, new(ContextSuite))
}

// makeContextWs creates a temp workspace with the given repos and groups.
// groups maps group name → path (relative to wsDir; empty string means no path).
// Writes .gitworkspace, calls changeToDir, returns wsDir.
func (s *ContextSuite) makeContextWs(repos []string, groups map[string]string) string {
	wsDir := s.T().TempDir()

	toml := "[workspace]\nname = \"test\"\n"
	for _, r := range repos {
		toml += fmt.Sprintf("\n[repos.%s]\npath = %q\n", r, r)
	}
	for gname, gpath := range groups {
		toml += fmt.Sprintf("\n[groups.%s]\nrepos = []\n", gname)
		if gpath != "" {
			toml += fmt.Sprintf("path = %q\n", gpath)
		}
	}

	s.Require().NoError(os.WriteFile(filepath.Join(wsDir, ".gitworkspace"), []byte(toml), 0o644))
	changeToDir(s.T(), wsDir)
	return wsDir
}

func (s *ContextSuite) TestContextShow() {
	cases := []struct {
		name      string
		localTOML string
		wantOut   string
	}{
		{"no context set", "", "(none)\n"},
		{"context set to web", "[context]\nactive = \"web\"\n", "web\n"},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.makeContextWs(nil, map[string]string{"web": ""})
			if tc.localTOML != "" {
				s.Require().NoError(os.WriteFile(
					filepath.Join(wsDir, ".gitworkspace.local"),
					[]byte(tc.localTOML), 0o644,
				))
			}
			out, err := execCmd(s.T(), "context")
			s.Require().NoError(err)
			s.Assert().Equal(tc.wantOut, out)
		})
	}
}

func (s *ContextSuite) TestContextSet() {
	cases := []struct {
		name    string
		group   string
		wantErr bool
	}{
		{"valid group", "web", false},
		{"unknown group", "nope", true},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.makeContextWs(nil, map[string]string{"web": ""})
			out, err := execCmd(s.T(), "context", tc.group)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Contains(out, tc.group)

			cfg, loadErr := config.Load(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(loadErr)
			s.Assert().Equal(tc.group, cfg.Context.Active)
		})
	}
}

func (s *ContextSuite) TestContextSet_WritesLocal() {
	wsDir := s.makeContextWs(nil, map[string]string{"web": ""})

	_, err := execCmd(s.T(), "context", "web")
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
	s.Require().NoError(err)
	s.Assert().Equal("web", cfg.Context.Active)
}

func (s *ContextSuite) TestContextClear() {
	wsDir := s.makeContextWs(nil, map[string]string{"web": ""})
	s.Require().NoError(os.WriteFile(
		filepath.Join(wsDir, ".gitworkspace.local"),
		[]byte("[context]\nactive = \"web\"\n"), 0o644,
	))

	_, err := execCmd(s.T(), "context", "none")
	s.Require().NoError(err)

	cfg, err := config.Load(filepath.Join(wsDir, ".gitworkspace"))
	s.Require().NoError(err)
	s.Assert().Equal("", cfg.Context.Active)
}

func (s *ContextSuite) TestContextAuto() {
	cases := []struct {
		name      string
		groups    map[string]string
		cwdSubdir string
		wantGroup string
		wantErr   bool
	}{
		{
			name:      "CWD under group path",
			groups:    map[string]string{"web": "apps"},
			cwdSubdir: "apps",
			wantGroup: "web",
		},
		{
			name:      "CWD not under any group path",
			groups:    map[string]string{"web": "apps"},
			cwdSubdir: "services",
			wantErr:   true,
		},
		{
			name:      "picks deepest group",
			groups:    map[string]string{"outer": "apps", "inner": "apps/sub"},
			cwdSubdir: "apps/sub",
			wantGroup: "inner",
		},
		{
			name:      "group without path is skipped",
			groups:    map[string]string{"web": ""},
			cwdSubdir: ".",
			wantErr:   true,
		},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			wsDir := s.makeContextWs(nil, tc.groups)

			targetDir := wsDir
			if tc.cwdSubdir != "." {
				targetDir = filepath.Join(wsDir, tc.cwdSubdir)
				s.Require().NoError(os.MkdirAll(targetDir, 0o755))
			}
			changeToDir(s.T(), targetDir)

			out, err := execCmd(s.T(), "context", "auto")
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Contains(out, tc.wantGroup)

			cfg, loadErr := config.Load(filepath.Join(wsDir, ".gitworkspace"))
			s.Require().NoError(loadErr)
			s.Assert().Equal(tc.wantGroup, cfg.Context.Active)
		})
	}
}
