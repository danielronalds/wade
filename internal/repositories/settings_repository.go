package repositories

type SettingsRepository struct{}

func NewSettingsRepository() SettingsRepository {
	return SettingsRepository{}
}

func (SettingsRepository) Load() (Settings, error) {
	return LoadSettings()
}

func (SettingsRepository) Save(settings Settings) error {
	return settings.Save()
}
