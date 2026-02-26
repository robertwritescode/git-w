package executor

import "bytes"

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

	lines := bytes.Split(b, []byte("\n"))
	endsWithNewline := b[len(b)-1] == '\n'
	if endsWithNewline {
		lines = lines[:len(lines)-1]
	}

	prefix := []byte("[" + name + "] ")
	var out []byte
	for i, line := range lines {
		if len(line) > 0 {
			out = append(out, prefix...)
			out = append(out, line...)
		}
		if i < len(lines)-1 || endsWithNewline {
			out = append(out, '\n')
		}
	}

	return out
}

func combinedOutput(r ExecResult) []byte {
	return append(r.Stdout, r.Stderr...)
}
