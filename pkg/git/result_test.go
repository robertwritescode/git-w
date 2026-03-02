package git

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/robertwritescode/git-w/pkg/testutil"
	"github.com/stretchr/testify/suite"
)

type ResultSuite struct {
	suite.Suite
}

func TestResult(t *testing.T) {
	testutil.RunSuite(t, new(ResultSuite))
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

func (s *ResultSuite) TestExecErrors() {
	sentinel := fmt.Errorf("network error")
	tests := []struct {
		name        string
		results     []ExecResult
		wantNil     bool
		wantContain string
	}{
		{
			name:    "all success",
			results: []ExecResult{{RepoName: "a", ExitCode: 0}, {RepoName: "b", ExitCode: 0}},
			wantNil: true,
		},
		{
			name:        "one non-zero exit",
			results:     []ExecResult{{RepoName: "a", ExitCode: 0}, {RepoName: "b", ExitCode: 1}},
			wantContain: "1 of 2",
		},
		{
			name:        "error field",
			results:     []ExecResult{{RepoName: "a", Err: sentinel}},
			wantContain: "network error",
		},
		{
			name:        "multiple failures",
			results:     []ExecResult{{RepoName: "a", ExitCode: 1}, {RepoName: "b", ExitCode: 2}},
			wantContain: "2 of 2",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := ExecErrors(tc.results)

			if tc.wantNil {
				s.Assert().NoError(err)
				return
			}

			s.Require().Error(err)
			s.Assert().Contains(err.Error(), tc.wantContain)
		})
	}
}

func (s *ResultSuite) TestWriteResults() {
	tests := []struct {
		name    string
		results []ExecResult
		want    string
	}{
		{
			name:    "empty",
			results: nil,
			want:    "",
		},
		{
			name:    "stdout only",
			results: []ExecResult{{Stdout: []byte("hello\n")}},
			want:    "hello\n",
		},
		{
			name:    "stderr only",
			results: []ExecResult{{Stderr: []byte("err\n")}},
			want:    "err\n",
		},
		{
			name:    "multiple results concatenated",
			results: []ExecResult{{Stdout: []byte("a\n")}, {Stdout: []byte("b\n")}},
			want:    "a\nb\n",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			var buf bytes.Buffer
			WriteResults(&buf, tc.results)
			s.Assert().Equal(tc.want, buf.String())
		})
	}
}
