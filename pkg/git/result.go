package git

import (
	"bytes"
	"fmt"
	"io"

	"github.com/robertwritescode/git-w/pkg/parallel"
)

// ExecResult holds the outcome of running a git command in one repo.
type ExecResult struct {
	RepoName string
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Err      error
}

// prefixLines prepends "[name] " to each non-empty line in b.
// Blank lines in the middle are preserved without a prefix.
// A trailing newline in b produces a trailing newline in the output.
func prefixLines(name string, b []byte) []byte {
	if len(b) == 0 {
		return nil
	}

	lines, endsWithNewline := splitTrailingNewline(b)
	prefix := []byte("[" + name + "] ")

	return joinPrefixed(lines, prefix, endsWithNewline)
}

func splitTrailingNewline(b []byte) ([][]byte, bool) {
	lines := bytes.Split(b, []byte("\n"))
	ends := b[len(b)-1] == '\n'
	if ends {
		lines = lines[:len(lines)-1]
	}
	return lines, ends
}

func joinPrefixed(lines [][]byte, prefix []byte, trailingNewline bool) []byte {
	var out []byte
	for i, line := range lines {
		if len(line) > 0 {
			out = append(out, prefix...)
			out = append(out, line...)
		}

		if i < len(lines)-1 || trailingNewline {
			out = append(out, '\n')
		}
	}
	return out
}

// WriteResults writes all result stdout and stderr to w.
func WriteResults(w io.Writer, results []ExecResult) {
	for _, r := range results {
		_, _ = w.Write(r.Stdout)
		_, _ = w.Write(r.Stderr)
	}
}

// ExecErrors returns a combined error if any results have non-zero exit codes or errors.
func ExecErrors(results []ExecResult) error {
	var failures []string

	for _, r := range results {
		if r.ExitCode != 0 || r.Err != nil {
			failures = append(failures, "  ["+r.RepoName+"]: "+failureMessage(r))
		}
	}

	return parallel.FormatFailureError(failures, len(results))
}

func failureMessage(r ExecResult) string {
	if r.Err != nil {
		return r.Err.Error()
	}
	return fmt.Sprintf("exit %d", r.ExitCode)
}
