package output

import (
	"fmt"
	"io"
)

// Writef writes formatted output and intentionally ignores write errors,
// which is suitable for terminal-style best-effort output.
func Writef(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}
