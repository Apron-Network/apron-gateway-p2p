package logger

type LogConfig struct {
	BaseDir    string `json:"base_dir"`
	Level      string `json:"level"`
	MaxSizeMb  int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAgeDay  int    `json:"max_age_day"`
}
