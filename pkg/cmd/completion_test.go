package cmd

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type CompletionSuite struct {
	testutil.CmdSuite
	wsDir string
}

func (s *CompletionSuite) SetupTest() {
	s.SetRoot(registerCompletion)
	s.wsDir = s.SetupWorkspaceDir()
}

func TestCompletionSuite(t *testing.T) {
	suite.Run(t, new(CompletionSuite))
}

func (s *CompletionSuite) TestCompletion() {
	tests := []struct {
		name  string
		shell string
	}{
		{name: "bash", shell: "bash"},
		{name: "zsh", shell: "zsh"},
		{name: "fish", shell: "fish"},
		{name: "powershell", shell: "powershell"},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			out, err := s.ExecuteCmd("completion", tt.shell)
			s.Require().NoError(err)
			s.Assert().NotEmpty(out)
		})
	}
}

func (s *CompletionSuite) TestCompletionErrorInvalidShell() {
	_, err := s.ExecuteCmd("completion", "fish-sauce")
	s.Require().Error(err)
}
