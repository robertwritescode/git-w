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
		s    func() string
		want int
	}{
		{
			name: "plain string",
			s:    func() string { return "hello" },
			want: 5,
		},
		{
			name: "empty string",
			s:    func() string { return "" },
			want: 0,
		},
		{
			name: "green colored",
			s:    func() string { return ColorInSync.Sprint("main ✓") },
			want: 6,
		},
		{
			name: "bold red",
			s:    func() string { return color.New(color.Bold, color.FgRed).Sprint("x") },
			want: 1,
		},
		{
			name: "no codes",
			s:    func() string { return "abc" },
			want: 3,
		},
		{
			name: "unicode symbol",
			s:    func() string { return "✓" },
			want: 1,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Assert().Equal(tt.want, visualWidth(tt.s()))
		})
	}
}
