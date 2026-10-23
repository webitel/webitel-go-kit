package rabbitmq

import "github.com/webitel/webitel-go-kit/pkg/logging"

const Component logging.Component = scopeName

func componentLogger(l *logging.Logger) *logging.Logger {
	if l == nil {
		l = logging.New()
	}

	return l.WithComponent(Component)
}
