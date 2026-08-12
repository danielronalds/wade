package settings

// Supported theme accent colour values.
const (
	ThemeAccentColorWhite  = "white"
	ThemeAccentColorOrange = "orange"
	ThemeAccentColorPurple = "purple"
)

// Agent configures one available agent terminal.
type Agent struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Default bool   `json:"default"`
} // @name Agent

// LinearSettings configures the optional Linear integration.
type LinearSettings struct {
	Enabled   bool   `json:"enabled"`
	Workspace string `json:"workspace"`
} // @name LinearSettings

// Settings is the detached editable user configuration stored on disk.
type Settings struct {
	WorkspaceDirectories               []string       `json:"workspaceDirectories"`
	Shell                              string         `json:"shell"`
	Agents                             []Agent        `json:"agents"`
	CopyIgnoredFilesOnWorktreeCreation bool           `json:"copyIgnoredFilesOnWorktreeCreation"`
	OpenWorktreesInNewTabs             bool           `json:"openWorktreesInNewTabs"`
	WorktreeCopyExcludes               []string       `json:"worktreeCopyExcludes"`
	ThemeAccentColor                   string         `json:"themeAccentColor" enums:"white,orange,purple"`
	Linear                             LinearSettings `json:"linear"`
} // @name Settings

// RuntimeConfiguration is the neutral resolved configuration used during startup and runtime reconfiguration.
type RuntimeConfiguration struct {
	Address                            string
	WorkspaceDirectoryPaths            []string
	WorkspaceDirectorySettings         []string
	Shell                              string
	Agents                             []Agent
	CopyIgnoredFilesOnWorktreeCreation bool
	WorktreeCopyExcludes               []string
	Linear                             LinearSettings
}

// UpdateResult contains persisted settings and their resolved runtime configuration.
type UpdateResult struct {
	Settings             Settings
	RuntimeConfiguration RuntimeConfiguration
}
