package agents_test

import (
	"testing"

	"github.com/robertwritescode/git-w/pkg/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrameworkFor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantMsg string
	}{
		{name: "known gsd", input: "gsd", wantErr: false},
		{name: "unknown value", input: "speckit", wantErr: true, wantMsg: "speckit"},
		{name: "empty string", input: "", wantErr: true, wantMsg: "gsd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := agents.FrameworkFor(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantMsg != "" {
					assert.Contains(t, err.Error(), tt.wantMsg)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.input, fw.Name())
		})
	}
}

func TestFrameworksFor(t *testing.T) {
	tests := []struct {
		name        string
		input       []string
		wantErr     bool
		wantLen     int
		errContains string
	}{
		{name: "nil input", input: nil, wantLen: 0},
		{name: "empty input", input: []string{}, wantLen: 0},
		{name: "single known", input: []string{"gsd"}, wantLen: 1},
		{name: "unknown value", input: []string{"gsd", "speckit"}, wantErr: true, errContains: "speckit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fws, err := agents.FrameworksFor(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Len(t, fws, tt.wantLen)
		})
	}
}

func TestGSDFrameworkName(t *testing.T) {
	fw := agents.GSDFramework{}
	assert.Equal(t, "gsd", fw.Name())
}

func TestGSDFrameworkWorkspaceCreationProhibited(t *testing.T) {
	fw := agents.GSDFramework{}
	assert.True(t, fw.WorkspaceCreationProhibited())
}

func TestGSDFrameworkProhibitedActionsNonEmpty(t *testing.T) {
	fw := agents.GSDFramework{}
	assert.NotEmpty(t, fw.ProhibitedActions())
}
