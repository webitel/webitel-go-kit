package rabbitmq

// Logger interface for logging inside the package.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, err error, args ...any)
}

// NoopLogger is a default no-operation logger.
type NoopLogger struct{}

func (n *NoopLogger) Info(msg string, args ...any)             {}
func (n *NoopLogger) Warn(msg string, args ...any)             {}
func (n *NoopLogger) Error(msg string, err error, args ...any) {}
