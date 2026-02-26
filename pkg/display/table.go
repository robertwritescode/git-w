package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/robertwritescode/git-w/pkg/repo"
)

// TableEntry is one row in the status table.
type TableEntry struct {
	Name        string
	Branch      string
	RemoteState repo.RemoteState
	Dirty       bool
	Staged      bool
	Untracked   bool
	Stashed     bool
	LastCommit  string
}

// RenderTable writes a formatted, color-coded status table to w.
// Each entry occupies one row. Columns are aligned by visual width.
func RenderTable(w io.Writer, entries []TableEntry) {
	rendered := renderEntries(entries)
	widths := columnWidths(rendered)

	writeHeader(w, widths)
	for _, r := range rendered {
		writeRow(w, r, widths)
	}
}

// writef writes formatted output, discarding write errors
// (appropriate for terminal I/O where write failures are unrecoverable).
func writef(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

type renderedEntry struct {
	name   string
	branch string
	status string
	commit string
}

func renderEntries(entries []TableEntry) []renderedEntry {
	out := make([]renderedEntry, len(entries))

	for i, e := range entries {
		out[i] = renderedEntry{
			name:   e.Name,
			branch: formatBranch(e.Branch, e.RemoteState),
			status: formatStatus(e),
			commit: e.LastCommit,
		}
	}

	return out
}

func columnWidths(rendered []renderedEntry) [4]int {
	widths := [4]int{4, 6, 6, 6} // REPO, BRANCH, STATUS, COMMIT

	for _, r := range rendered {
		cols := [4]int{
			visualWidth(r.name),
			visualWidth(r.branch),
			visualWidth(r.status),
			visualWidth(r.commit),
		}
		for i, w := range cols {
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	return widths
}

func writeHeader(w io.Writer, widths [4]int) {
	writef(w, "%s  %s  %s  %s\n",
		padTo("REPO", widths[0]),
		padTo("BRANCH", widths[1]),
		padTo("STATUS", widths[2]),
		"COMMIT",
	)
}

func writeRow(w io.Writer, r renderedEntry, widths [4]int) {
	writef(w, "%s  %s  %s  %s\n",
		padTo(r.name, widths[0]),
		padTo(r.branch, widths[1]),
		padTo(r.status, widths[2]),
		r.commit,
	)
}

func formatBranch(branch string, state repo.RemoteState) string {
	var symbol string
	var c = colorNoRemote

	switch state {
	case repo.InSync:
		symbol, c = "✓", colorInSync
	case repo.LocalAhead:
		symbol, c = "↑", colorAhead
	case repo.RemoteAhead:
		symbol, c = "↓", colorBehind
	case repo.Diverged:
		symbol, c = "⇕", colorDiverged
	default:
		symbol = "∅"
	}

	return c.Sprint(branch + " " + symbol)
}

func formatStatus(e TableEntry) string {
	var b strings.Builder
	if e.Dirty {
		b.WriteByte('*')
	}
	if e.Staged {
		b.WriteByte('+')
	}
	if e.Untracked {
		b.WriteByte('?')
	}
	if e.Stashed {
		b.WriteByte('$')
	}
	return b.String()
}

func padTo(s string, width int) string {
	padding := width - visualWidth(s)
	if padding <= 0 {
		return s
	}

	return s + strings.Repeat(" ", padding)
}
