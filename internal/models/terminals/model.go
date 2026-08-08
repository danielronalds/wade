package terminals

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"

	"wade/internal/infrastructure/pty"
)

// Model owns terminal resources, processes, buffering, clients, and live sessions.
type Model struct {
	workspaces WorkspaceDiscovery
	pty        PTY

	mu                    sync.Mutex
	configuration         Configuration
	terminals             map[string]*terminalProcess
	selectedAgentTerminal map[string]*terminalProcess
}

// New constructs an application-scoped Terminals Model.
func New(workspaces WorkspaceDiscovery, pty PTY, configuration Configuration) *Model {
	configuration.Agents = cloneAgents(configuration.Agents)
	return &Model{
		workspaces:            workspaces,
		pty:                   pty,
		configuration:         configuration,
		terminals:             make(map[string]*terminalProcess),
		selectedAgentTerminal: make(map[string]*terminalProcess),
	}
}

// Configure atomically updates configuration used by future terminals.
func (model *Model) Configure(configuration Configuration) {
	model.mu.Lock()
	defer model.mu.Unlock()
	configuration.ServerAddress = model.configuration.ServerAddress
	configuration.Agents = cloneAgents(configuration.Agents)
	model.configuration = configuration
}

// Put idempotently starts or returns one terminal resource.
func (model *Model) Put(ctx context.Context, workspaceID string, terminalID string) (Terminal, bool, error) {
	workspacePath, err := model.workspacePath(workspaceID)
	if err != nil {
		return Terminal{}, false, err
	}
	if err := ctx.Err(); err != nil {
		return Terminal{}, false, err
	}

	model.mu.Lock()
	defer model.mu.Unlock()

	descriptor, err := resolveTerminalDescriptor(terminalID, model.configuration.Agents)
	if err != nil {
		return Terminal{}, false, err
	}
	key := terminalKey(workspaceID, descriptor.id)
	if terminal, found := model.terminals[key]; found && !terminal.isClosed() {
		return terminal.snapshot(), false, nil
	}
	delete(model.terminals, key)

	environment := pty.WadeEnvironment{
		WorkspaceID: workspaceID,
		TerminalID:  descriptor.id,
		Address:     model.configuration.ServerAddress,
	}
	process, err := model.startProcess(workspacePath, environment, descriptor)
	if err != nil {
		return Terminal{}, false, err
	}

	resource := Terminal{
		ID:          descriptor.id,
		WorkspaceID: workspaceID,
		Role:        descriptor.role,
		Agent:       cloneString(descriptor.agent),
		Status:      TerminalStatusRunning,
		SocketURL:   terminalSocketURL(workspaceID, descriptor.id),
	}
	terminal := &terminalProcess{
		resource: resource,
		key:      key,
		manager:  model,
		process:  process,
		buffer:   newOutputBuffer(terminalBufferBytes),
		clients:  make(map[*terminalClient]struct{}),
	}
	model.terminals[key] = terminal
	go terminal.readLoop()
	return terminal.snapshot(), true, nil
}

// Get returns one detached terminal resource.
func (model *Model) Get(ctx context.Context, workspaceID string, terminalID string) (Terminal, error) {
	if _, err := model.workspacePath(workspaceID); err != nil {
		return Terminal{}, err
	}
	if err := ctx.Err(); err != nil {
		return Terminal{}, err
	}

	model.mu.Lock()
	defer model.mu.Unlock()
	terminal, err := model.lookupLocked(workspaceID, terminalID)
	if err != nil {
		return Terminal{}, err
	}
	return terminal.snapshot(), nil
}

// List returns sorted detached terminal resources for one workspace.
func (model *Model) List(ctx context.Context, workspaceID string) ([]Terminal, error) {
	if _, err := model.workspacePath(workspaceID); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	model.mu.Lock()
	defer model.mu.Unlock()
	prefix := workspaceID + "\x00"
	terminals := make([]Terminal, 0)
	for key, terminal := range model.terminals {
		if strings.HasPrefix(key, prefix) && !terminal.isClosed() {
			terminals = append(terminals, terminal.snapshot())
		}
	}
	sort.Slice(terminals, func(firstIndex int, secondIndex int) bool {
		return terminals[firstIndex].ID < terminals[secondIndex].ID
	})
	return terminals, nil
}

// Delete closes and removes one exact terminal.
func (model *Model) Delete(ctx context.Context, workspaceID string, terminalID string) error {
	if _, err := model.workspacePath(workspaceID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	model.mu.Lock()
	terminal, err := model.lookupLocked(workspaceID, terminalID)
	if err != nil {
		model.mu.Unlock()
		return err
	}
	delete(model.terminals, terminal.key)
	if model.selectedAgentTerminal[workspaceID] == terminal {
		delete(model.selectedAgentTerminal, workspaceID)
	}
	model.mu.Unlock()
	terminal.close()
	return nil
}

// DeleteAll closes all terminals for a workspace.
func (model *Model) DeleteAll(ctx context.Context, workspaceID string) (int, error) {
	if _, err := model.workspacePath(workspaceID); err != nil {
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	model.mu.Lock()
	prefix := workspaceID + "\x00"
	workspaceTerminals := make([]*terminalProcess, 0)
	for key, terminal := range model.terminals {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		delete(model.terminals, key)
		if !terminal.isClosed() {
			workspaceTerminals = append(workspaceTerminals, terminal)
		}
	}
	delete(model.selectedAgentTerminal, workspaceID)
	model.mu.Unlock()

	for _, terminal := range workspaceTerminals {
		terminal.close()
	}
	return len(workspaceTerminals), nil
}

// Input validates and writes terminal input.
func (model *Model) Input(ctx context.Context, input Input) error {
	if input.Text == "" {
		return TerminalInputRequiredError{}
	}
	if input.Mode != InputModeRaw && input.Mode != InputModeBracketedPaste {
		return InvalidInputModeError{Mode: input.Mode}
	}

	session, err := model.Connect(ctx, input.WorkspaceID, input.TerminalID)
	if err != nil {
		return err
	}
	defer session.Close()

	data := []byte(input.Text)
	if input.Mode == InputModeBracketedPaste {
		data = []byte("\x1b[200~" + input.Text + "\x1b[201~")
	}
	bytesWritten, err := session.Write(data)
	if err == nil && bytesWritten != len(data) {
		return io.ErrShortWrite
	}
	return err
}

// Connect returns an explicit live session for an existing terminal.
func (model *Model) Connect(ctx context.Context, workspaceID string, terminalID string) (*TerminalSession, error) {
	if _, err := model.workspacePath(workspaceID); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	model.mu.Lock()
	defer model.mu.Unlock()
	terminal, err := model.lookupLocked(workspaceID, terminalID)
	if err != nil {
		return nil, err
	}
	return terminal.attach(), nil
}

// ActiveTerminalCount returns the current process count for a workspace.
func (model *Model) ActiveTerminalCount(workspaceID string) int {
	model.mu.Lock()
	defer model.mu.Unlock()
	count := 0
	prefix := workspaceID + "\x00"
	for key, terminal := range model.terminals {
		if strings.HasPrefix(key, prefix) && !terminal.isClosed() {
			count++
		}
	}
	return count
}

// ActiveWorkspaceIDs returns sorted unique workspace IDs with live terminals.
func (model *Model) ActiveWorkspaceIDs() []string {
	model.mu.Lock()
	defer model.mu.Unlock()
	workspaceIDs := make([]string, 0, len(model.terminals))
	for _, terminal := range model.terminals {
		if !terminal.isClosed() {
			workspaceIDs = append(workspaceIDs, terminal.resource.WorkspaceID)
		}
	}
	sort.Strings(workspaceIDs)
	return slices.Compact(workspaceIDs)
}

// Close closes every terminal owned by the Model.
func (model *Model) Close() {
	model.mu.Lock()
	activeTerminals := make([]*terminalProcess, 0, len(model.terminals))
	for _, terminal := range model.terminals {
		activeTerminals = append(activeTerminals, terminal)
	}
	model.terminals = make(map[string]*terminalProcess)
	model.selectedAgentTerminal = make(map[string]*terminalProcess)
	model.mu.Unlock()

	for _, terminal := range activeTerminals {
		terminal.close()
	}
}

func (model *Model) workspacePath(workspaceID string) (string, error) {
	workspacePath, found, err := model.workspaces.Resolve(workspaceID)
	if err != nil {
		return "", fmt.Errorf("resolving workspace %q: %w", workspaceID, err)
	}
	if !found {
		return "", WorkspaceNotFoundError{WorkspaceID: workspaceID}
	}
	return workspacePath, nil
}

func (model *Model) startProcess(workspacePath string, environment pty.WadeEnvironment, descriptor terminalDescriptor) (pty.Process, error) {
	size := pty.Size{Cols: 80, Rows: 24}
	if descriptor.role == TerminalRoleAgent && descriptor.command != "" {
		return model.pty.StartCommand(model.configuration.Shell, workspacePath, environment, descriptor.command, size)
	}
	return model.pty.StartInteractive(model.configuration.Shell, workspacePath, environment, size)
}

func (model *Model) lookupLocked(workspaceID string, terminalID string) (*terminalProcess, error) {
	descriptor, err := resolveTerminalDescriptor(terminalID, model.configuration.Agents)
	if err != nil {
		return nil, err
	}
	terminal, found := model.terminals[terminalKey(workspaceID, descriptor.id)]
	if !found || terminal.isClosed() {
		return nil, TerminalNotFoundError{WorkspaceID: workspaceID, TerminalID: descriptor.id}
	}
	return terminal, nil
}

func (model *Model) remove(key string, terminal *terminalProcess) {
	model.mu.Lock()
	defer model.mu.Unlock()
	if model.terminals[key] == terminal {
		delete(model.terminals, key)
	}
	workspaceID := terminal.resource.WorkspaceID
	if model.selectedAgentTerminal[workspaceID] == terminal {
		delete(model.selectedAgentTerminal, workspaceID)
	}
}

func (model *Model) activateAgent(terminal *terminalProcess) {
	model.mu.Lock()
	defer model.mu.Unlock()
	workspaceID := terminal.resource.WorkspaceID
	if terminal.resource.Role == TerminalRoleAgent && model.terminals[terminal.key] == terminal && !terminal.isClosed() {
		model.selectedAgentTerminal[workspaceID] = terminal
	}
}

func terminalKey(workspaceID string, terminalID string) string {
	return workspaceID + "\x00" + terminalID
}

func terminalSocketURL(workspaceID string, terminalID string) string {
	return fmt.Sprintf("/api/v1/workspaces/%s/terminals/%s/socket", url.PathEscape(workspaceID), url.PathEscape(terminalID))
}

func cloneAgents(agents []Agent) []Agent {
	return append([]Agent(nil), agents...)
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
