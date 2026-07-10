package config

// TODO: Review properly

import "wade/internal/repositories"

type Agent = repositories.Agent

type Settings = repositories.Settings

const (
	ThemeAccentColorWhite  = repositories.ThemeAccentColorWhite
	ThemeAccentColorOrange = repositories.ThemeAccentColorOrange
	ThemeAccentColorPurple = repositories.ThemeAccentColorPurple
)
