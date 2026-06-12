package appconfig

// Log holds structured logging configuration.
// Env vars: LOG_LEVEL, LOG_JSON, LOG_OTEL, LOG_FILE, LOG_CONSOLE.
type Log struct {
	Level   string `mapstructure:"level"`
	JSON    bool   `mapstructure:"json"`
	Otel    bool   `mapstructure:"otel"`
	File    string `mapstructure:"file"`
	Console bool   `mapstructure:"console"`
}
