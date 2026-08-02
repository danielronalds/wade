package config

import "fmt"

type InvalidSettingsError struct {
	Err error
}

func (e InvalidSettingsError) Error() string {
	return fmt.Sprintf("invalid settings: %v", e.Err)
}

func (e InvalidSettingsError) Unwrap() error {
	return e.Err
}
