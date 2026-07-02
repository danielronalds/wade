package manager

import "testing"

func TestShouldStartAgentPaneCommandOnlyForAgentTerminal(t *testing.T) {
	tests := map[string]struct {
		terminalName     string
		agentPaneCommand string
		want             bool
	}{
		"agent terminal with command": {
			terminalName:     agentTerminalName,
			agentPaneCommand: "pi -c",
			want:             true,
		},
		"agent terminal without command": {
			terminalName:     agentTerminalName,
			agentPaneCommand: "",
			want:             false,
		},
		"misc terminal with command": {
			terminalName:     "misc",
			agentPaneCommand: "pi -c",
			want:             false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := shouldStartAgentPaneCommand(test.terminalName, test.agentPaneCommand)
			if got != test.want {
				t.Fatalf("shouldStartAgentPaneCommand() = %v, want %v", got, test.want)
			}
		})
	}
}
