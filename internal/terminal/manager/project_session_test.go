package manager

import "testing"

func TestShouldStartAgentCommandOnlyForAgentTerminal(t *testing.T) {
	tests := map[string]struct {
		terminalName string
		agentCommand string
		want         bool
	}{
		"agent terminal with command": {
			terminalName: agentTerminalName,
			agentCommand: "pi -c",
			want:         true,
		},
		"agent terminal without command": {
			terminalName: agentTerminalName,
			agentCommand: "",
			want:         false,
		},
		"misc terminal with command": {
			terminalName: "misc",
			agentCommand: "pi -c",
			want:         false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := shouldStartAgentCommand(test.terminalName, test.agentCommand)
			if got != test.want {
				t.Fatalf("shouldStartAgentCommand() = %v, want %v", got, test.want)
			}
		})
	}
}
