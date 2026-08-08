package settings

// FileSystem provides settings-file access and executable inspection.
type FileSystem interface {
	SettingsFilePath(homeDirectory string) string
	SettingsFileExists(homeDirectory string) (bool, error)
	ReadSettingsFile(homeDirectory string) ([]byte, error)
	WriteSettingsFile(homeDirectory string, contents []byte) error
	IsExecutableFile(path string) (bool, error)
}

// Environment provides process environment values used to resolve settings.
type Environment interface {
	HomeDirectory() (string, error)
	Variable(name string) string
	InheritedShell() string
	LookPath(name string) (string, error)
}
