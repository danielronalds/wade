package terminals

import (
	"errors"
	"testing"
)

func TestResolveTerminalDescriptor(t *testing.T) {
	agents := []Agent{{Name: "Pi", Command: "pi -c", Default: true}}
	tests := map[string]struct {
		terminalID string
		wantID     string
		wantRole   TerminalRole
		wantError  bool
	}{
		"misc":                   {terminalID: "misc", wantID: "misc", wantRole: TerminalRoleMisc},
		"server":                 {terminalID: "server", wantID: "server", wantRole: TerminalRoleServer},
		"scratchpad":             {terminalID: "scratchpad", wantID: "scratchpad", wantRole: TerminalRoleScratchpad},
		"case-insensitive agent": {terminalID: "agent:PI", wantID: "agent:pi", wantRole: TerminalRoleAgent},
		"unknown agent":          {terminalID: "agent:claude", wantError: true},
		"unknown role":           {terminalID: "terminal", wantError: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			descriptor, err := resolveTerminalDescriptor(test.terminalID, agents)
			if !test.wantError {
				if err != nil {
					t.Fatalf("resolveTerminalDescriptor() error = %v, want nil", err)
				}
				if descriptor.id != test.wantID || descriptor.role != test.wantRole {
					t.Fatalf("descriptor = %#v, want ID %q and role %q", descriptor, test.wantID, test.wantRole)
				}
				if test.wantRole == TerminalRoleAgent && (descriptor.agent == nil || *descriptor.agent != "Pi") {
					t.Fatalf("descriptor agent = %#v, want configured display name Pi", descriptor.agent)
				}
				return
			}

			var invalidIDError InvalidTerminalIDError
			var agentError AgentNotConfiguredError
			if !errors.As(err, &invalidIDError) && !errors.As(err, &agentError) {
				t.Fatalf("resolveTerminalDescriptor() error = %v, want typed validation error", err)
			}
		})
	}
}
