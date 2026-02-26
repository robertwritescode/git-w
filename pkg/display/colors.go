package display

import (
	"regexp"

	"github.com/fatih/color"
)

var (
	colorInSync   = color.New(color.FgGreen)
	colorAhead    = color.New(color.FgHiMagenta)
	colorBehind   = color.New(color.FgYellow)
	colorDiverged = color.New(color.FgRed)
	colorNoRemote = color.New(color.FgWhite)
)

// ansiEscape matches ANSI CSI escape sequences used by fatih/color.
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visualWidth returns the visible character width of s, ignoring ANSI codes.
func visualWidth(s string) int {
	plain := ansiEscape.ReplaceAllString(s, "")
	return len([]rune(plain))
}
