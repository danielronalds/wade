package settings

import "fmt"

// InvalidSettingsError reports settings that cannot be normalised or resolved.
type InvalidSettingsError struct {
	Err error
}

func (err InvalidSettingsError) Error() string {
	return fmt.Sprintf("invalid settings: %v", err.Err)
}

func (err InvalidSettingsError) Unwrap() error {
	return err.Err
}
