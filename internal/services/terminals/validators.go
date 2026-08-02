package terminals

import "strings"

func resolveTerminalDescriptor(terminalID string, agents []Agent) (terminalDescriptor, error) {
	terminalID = strings.ToLower(strings.TrimSpace(terminalID))
	switch terminalID {
	case string(TerminalRoleMisc):
		return terminalDescriptor{id: terminalID, role: TerminalRoleMisc}, nil
	case string(TerminalRoleServer):
		return terminalDescriptor{id: terminalID, role: TerminalRoleServer}, nil
	case string(TerminalRoleScratchpad):
		return terminalDescriptor{id: terminalID, role: TerminalRoleScratchpad}, nil
	}

	agentName, isAgent := strings.CutPrefix(terminalID, "agent:")
	if !isAgent || agentName == "" {
		return terminalDescriptor{}, InvalidTerminalIDError{TerminalID: terminalID}
	}

	for _, agent := range agents {
		if !strings.EqualFold(agent.Name, agentName) {
			continue
		}

		displayName := agent.Name
		return terminalDescriptor{
			id:      "agent:" + strings.ToLower(agent.Name),
			role:    TerminalRoleAgent,
			agent:   &displayName,
			command: agent.Command,
		}, nil
	}

	return terminalDescriptor{}, AgentNotConfiguredError{AgentName: agentName}
}
