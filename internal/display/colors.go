package display

import (
	"regexp"

	"github.com/fatih/color"
)

// ColorInSync is used for repos that are in sync with their remote.
var ColorInSync = color.New(color.FgGreen)

// ColorAhead is used for repos with local commits not yet pushed.
var ColorAhead = color.New(color.FgHiMagenta)

// ColorBehind is used for repos with remote commits not yet pulled.
var ColorBehind = color.New(color.FgYellow)

// ColorDiverged is used for repos that have diverged from their remote.
var ColorDiverged = color.New(color.FgRed)

// ColorNoRemote is used for repos with no configured remote.
var ColorNoRemote = color.New(color.FgWhite)

// ansiEscape matches ANSI CSI escape sequences used by fatih/color.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visualWidth returns the visible character width of s, ignoring ANSI codes.
func visualWidth(s string) int {
	plain := ansiEscape.ReplaceAllString(s, "")
	return len([]rune(plain))
}
