package display

import (
	"fmt"
	"io"
	"strings"

	"github.com/robertwritescode/git-workspace/internal/repo"
)

// TableEntry pairs a repo name with its current status for table rendering.
type TableEntry struct {
	Name   string
	Status repo.RepoStatus
}

// RenderTable writes a formatted, color-coded status table to w.
// Each entry occupies one row. Columns are aligned by visual width.
func RenderTable(w io.Writer, entries []TableEntry) {
	widths := columnWidths(entries)
	writeHeader(w, widths)
	for _, e := range entries {
		writeRow(w, e, widths)
	}
}

func columnWidths(entries []TableEntry) [4]int {
	widths := [4]int{4, 6, 6, 6} // REPO, BRANCH, STATUS, COMMIT
	for _, e := range entries {
		branch := formatBranch(e.Status.Branch, e.Status.RemoteState)
		status := formatStatus(e.Status)
		cols := [4]int{
			visualWidth(e.Name),
			visualWidth(branch),
			visualWidth(status),
			visualWidth(e.Status.LastCommit),
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
	fmt.Fprintf(w, "%s  %s  %s  %s\n",
		padTo("REPO", widths[0]),
		padTo("BRANCH", widths[1]),
		padTo("STATUS", widths[2]),
		"COMMIT",
	)
}

func writeRow(w io.Writer, e TableEntry, widths [4]int) {
	branch := formatBranch(e.Status.Branch, e.Status.RemoteState)
	status := formatStatus(e.Status)
	fmt.Fprintf(w, "%s  %s  %s  %s\n",
		padTo(e.Name, widths[0]),
		padTo(branch, widths[1]),
		padTo(status, widths[2]),
		e.Status.LastCommit,
	)
}

func formatBranch(branch string, state repo.RemoteState) string {
	var symbol string
	var c = ColorNoRemote
	switch state {
	case repo.InSync:
		symbol, c = "✓", ColorInSync
	case repo.LocalAhead:
		symbol, c = "↑", ColorAhead
	case repo.RemoteAhead:
		symbol, c = "↓", ColorBehind
	case repo.Diverged:
		symbol, c = "⇕", ColorDiverged
	default:
		symbol = "∅"
	}
	return c.Sprint(branch + " " + symbol)
}

func formatStatus(s repo.RepoStatus) string {
	var b strings.Builder
	if s.Dirty {
		b.WriteByte('*')
	}
	if s.Staged {
		b.WriteByte('+')
	}
	if s.Untracked {
		b.WriteByte('?')
	}
	if s.Stashed {
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
