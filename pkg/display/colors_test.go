package display

import (
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/suite"
)

type ColorsSuite struct {
	suite.Suite
}

func TestColorsSuite(t *testing.T) {
	suite.Run(t, new(ColorsSuite))
}

func (s *ColorsSuite) SetupTest() {
	color.NoColor = true
}

func (s *ColorsSuite) TestVisualWidth() {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"plain string", "hello", 5},
		{"empty string", "", 0},
		{"green colored", colorInSync.Sprint("main ✓"), 6},
		{"bold red", color.New(color.Bold, color.FgRed).Sprint("x"), 1},
		{"no codes", "abc", 3},
		{"unicode symbol", "✓", 1},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.want, visualWidth(tt.s))
		})
	}
}
