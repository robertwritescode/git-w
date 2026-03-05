package display

import (
	"io"
	"strings"

	"github.com/robertwritescode/git-w/pkg/config"
	"github.com/robertwritescode/git-w/pkg/output"
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

// WorktreeSet groups entries belonging to a single worktree set.
type WorktreeSet struct {
	SetName  string
	Branches []string
}

// WorkgroupSection holds entries for one workgroup in the workgroup table.
type WorkgroupSection struct {
	Name    string
	Entries []TableEntry
}

// groupedRow is one visual row in the grouped table output.
type groupedRow struct {
	renderedEntry
	isSetHeader bool
}

// workgroupRow is one row in the workgroup table.
type workgroupRow struct {
	workgroup string
	renderedEntry
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

// RenderGroupedTable renders entries with worktree sets collapsed under header rows.
func RenderGroupedTable(w io.Writer, entries []TableEntry, sets []WorktreeSet) {
	standalone, grouped := splitBySet(entries, sets)
	rows := buildGroupedRows(standalone, grouped, sets)
	widths := groupedColumnWidths(rows)

	writeHeader(w, widths)
	writeGroupedRows(w, rows, widths)
}

func groupedColumnWidths(rows []groupedRow) [4]int {
	entries := make([]renderedEntry, len(rows))
	for i, r := range rows {
		entries[i] = r.renderedEntry
	}

	return columnWidths(entries)
}

func writeGroupedRows(w io.Writer, rows []groupedRow, widths [4]int) {
	for _, r := range rows {
		writeGroupedRow(w, r, widths)
	}
}

func writeGroupedRow(w io.Writer, r groupedRow, widths [4]int) {
	if r.isSetHeader {
		output.Writef(w, "%s\n", padTo(r.name, widths[0]))
		return
	}

	writeRow(w, r.renderedEntry, widths)
}

func splitBySet(entries []TableEntry, sets []WorktreeSet) ([]TableEntry, map[string][]TableEntry) {
	setMemberLookup := make(map[string]string)
	for _, s := range sets {
		for _, branch := range s.Branches {
			repoName := config.WorktreeRepoName(s.SetName, branch)
			setMemberLookup[repoName] = s.SetName
		}
	}

	standalone := make([]TableEntry, 0, len(entries))
	grouped := make(map[string][]TableEntry)

	for _, e := range entries {
		if setName, ok := setMemberLookup[e.Name]; ok {
			grouped[setName] = append(grouped[setName], e)
		} else {
			standalone = append(standalone, e)
		}
	}

	return standalone, grouped
}

func buildGroupedRows(standalone []TableEntry, grouped map[string][]TableEntry, sets []WorktreeSet) []groupedRow {
	renderedStandalone := renderEntries(standalone)
	rows := make([]groupedRow, 0, len(standalone)+len(grouped)*2)

	iStandalone := 0
	iSet := 0
	for iStandalone < len(standalone) || iSet < len(sets) {
		iSet = skipEmptySets(sets, grouped, iSet)

		if pickStandalone(standalone, sets, iStandalone, iSet) {
			rows = append(rows, groupedRow{renderedEntry: renderedStandalone[iStandalone]})
			iStandalone++
			continue
		}

		if iSet >= len(sets) {
			break
		}

		rows = appendSetRows(rows, sets[iSet], grouped)
		iSet++
	}

	return rows
}

func skipEmptySets(sets []WorktreeSet, grouped map[string][]TableEntry, i int) int {
	for i < len(sets) && len(grouped[sets[i].SetName]) == 0 {
		i++
	}

	return i
}

func pickStandalone(standalone []TableEntry, sets []WorktreeSet, iStandalone, iSet int) bool {
	if iStandalone >= len(standalone) {
		return false
	}

	if iSet >= len(sets) {
		return true
	}

	return standalone[iStandalone].Name < sets[iSet].SetName
}

func appendSetRows(rows []groupedRow, set WorktreeSet, grouped map[string][]TableEntry) []groupedRow {
	rows = append(rows, groupedRow{
		renderedEntry: renderedEntry{name: set.SetName},
		isSetHeader:   true,
	})

	entries := grouped[set.SetName]
	renderedSetEntries := renderEntries(entries)

	for i, r := range renderedSetEntries {
		r.name = "  └ " + branchDisplayName(set.SetName, entries[i].Name)
		rows = append(rows, groupedRow{renderedEntry: r})
	}

	return rows
}

func branchDisplayName(setName, entryName string) string {
	return strings.TrimPrefix(entryName, setName+"-")
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
	widths := [4]int{4, 6, 6, 6}

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
	output.Writef(w, "%s  %s  %s  %s\n",
		padTo("REPO", widths[0]),
		padTo("BRANCH", widths[1]),
		padTo("STATUS", widths[2]),
		"COMMIT",
	)
}

func writeRow(w io.Writer, r renderedEntry, widths [4]int) {
	output.Writef(w, "%s  %s  %s  %s\n",
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

// RenderWorkgroupTable renders a 5-column workgroup status table.
func RenderWorkgroupTable(w io.Writer, sections []WorkgroupSection) {
	if len(sections) == 0 {
		return
	}

	rows := renderWorkgroupRows(sections)
	widths := workgroupColumnWidths(rows)

	writeWorkgroupHeader(w, widths)
	for _, r := range rows {
		writeWorkgroupRow(w, r, widths)
	}
}

func renderWorkgroupRows(sections []WorkgroupSection) []workgroupRow {
	rows := make([]workgroupRow, 0)

	for _, section := range sections {
		rendered := renderEntries(section.Entries)
		for i, r := range rendered {
			wgName := ""
			if i == 0 {
				wgName = section.Name
			}
			rows = append(rows, workgroupRow{
				workgroup:     wgName,
				renderedEntry: r,
			})
		}
	}

	return rows
}

func workgroupColumnWidths(rows []workgroupRow) [5]int {
	widths := [5]int{9, 4, 6, 6, 6}

	for _, r := range rows {
		cols := [5]int{
			visualWidth(r.workgroup),
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

func writeWorkgroupHeader(w io.Writer, widths [5]int) {
	output.Writef(w, "%s  %s  %s  %s  %s\n",
		padTo("WORKGROUP", widths[0]),
		padTo("REPO", widths[1]),
		padTo("BRANCH", widths[2]),
		padTo("STATUS", widths[3]),
		"COMMIT",
	)
}

func writeWorkgroupRow(w io.Writer, r workgroupRow, widths [5]int) {
	output.Writef(w, "%s  %s  %s  %s  %s\n",
		padTo(r.workgroup, widths[0]),
		padTo(r.name, widths[1]),
		padTo(r.branch, widths[2]),
		padTo(r.status, widths[3]),
		r.commit,
	)
}
