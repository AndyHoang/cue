package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIHelp(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedStatus int
		contains       []string
	}{
		{
			name:           "Help subcommand",
			args:           []string{"help"},
			expectedStatus: 0,
			contains:       []string{"Usage of cue:", "completion", "help"},
		},
		{
			name:           "Unknown command",
			args:           []string{"unknown"},
			expectedStatus: 1,
			contains:       []string{"Error: unknown command \"unknown\""},
		},
		{
			name:           "Completion bash",
			args:           []string{"completion", "bash"},
			expectedStatus: 0,
			contains:       []string{"_cue_completions()"},
		},
		{
			name:           "Completion zsh",
			args:           []string{"completion", "zsh"},
			expectedStatus: 0,
			contains:       []string{"#compdef cue", "_cue()"},
		},
		{
			name:           "Completion fish",
			args:           []string{"completion", "fish"},
			expectedStatus: 0,
			contains:       []string{"function __fish_cue_no_subcommand"},
		},
		{
			name:           "Completion powershell",
			args:           []string{"completion", "powershell"},
			expectedStatus: 0,
			contains:       []string{"Register-ArgumentCompleter"},
		},
		{
			name:           "Completion usage",
			args:           []string{"completion"},
			expectedStatus: 1,
			contains:       []string{"Usage: cue completion [bash|zsh|fish|powershell]"},
		},
		{
			name:           "Version flag short",
			args:           []string{"-v"},
			expectedStatus: 0,
			contains:       []string{"cue dev"},
		},
		{
			name:           "Version flag long",
			args:           []string{"-version"},
			expectedStatus: 0,
			contains:       []string{"cue dev"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			status := runCLI(tt.args, stdout, stderr)

			if status != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, status)
			}

			output := stdout.String() + stderr.String()
			for _, s := range tt.contains {
				if !strings.Contains(output, s) {
					t.Errorf("expected output to contain %q, but it didn't.\nOutput:\n%s", s, output)
				}
			}
		})
	}
}
