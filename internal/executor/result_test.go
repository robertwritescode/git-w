package executor

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ResultSuite struct {
	suite.Suite
}

func TestResult(t *testing.T) {
	suite.Run(t, new(ResultSuite))
}

func (s *ResultSuite) TestPrefixLines() {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{"empty input", nil, nil},
		{"single line", []byte("hello\n"), []byte("[repo] hello\n")},
		{"multi-line", []byte("a\nb\n"), []byte("[repo] a\n[repo] b\n")},
		{"no trailing newline", []byte("hello"), []byte("[repo] hello")},
		{"blank middle line", []byte("a\n\nb\n"), []byte("[repo] a\n\n[repo] b\n")},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Assert().Equal(tc.want, prefixLines("repo", tc.input))
		})
	}
}

func (s *ResultSuite) TestCombinedOutput() {
	tests := []struct {
		name   string
		stdout []byte
		stderr []byte
		want   []byte
	}{
		{"stdout only", []byte("out"), nil, []byte("out")},
		{"stderr only", nil, []byte("err"), []byte("err")},
		{"both", []byte("o"), []byte("e"), []byte("oe")},
		{"both empty", nil, nil, nil},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			r := ExecResult{Stdout: tc.stdout, Stderr: tc.stderr}
			s.Assert().Equal(tc.want, combinedOutput(r))
		})
	}
}

func (s *ResultSuite) TestExecResult_NonZeroExit() {
	r := ExecResult{RepoName: "repo", ExitCode: 1}
	s.Assert().Equal(1, r.ExitCode)
}
