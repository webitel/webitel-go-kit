package discovery

import (
	"context"
	"fmt"
	"time"
)

type DiscoveryProvider interface {
	Registrar
	Discovery
	DefaultOptionConfigurable
	KV() KVProvider
	ListServices() map[string][]*ServiceInstance
}

type Registrar interface {
	Register(ctx context.Context, instance *ServiceInstance) error
	Deregister(ctx context.Context, instance *ServiceInstance) error
}

type Discovery interface {
	GetService(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	GetWatcher(ctx context.Context, serviceName string) (Watcher, error)
}

type Watcher interface {
	Next() ([]*ServiceInstance, error)
	Stop() error
}

type KVProvider interface {
	PutToKV(ctx context.Context, key string, value []byte) error
	GetFromKV(ctx context.Context, key string) ([]byte, error)
	DeleteFromKV(ctx context.Context, key string) error

	GetKVWatcher(ctx context.Context, key string) KVWatcher
}

type KVWatcher interface {
	Next() ([]byte, error)
	Stop() error
}

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(string, ...any)
}

type ServiceInstance struct {
	Id        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Metadata  map[string]string `json:"metadata"`
	Endpoints []string          `json:"endpoints"`
}

// String returns a string representation of the ServiceInstance in the format "<name>-<id>".
func (s *ServiceInstance) String() string {
	return fmt.Sprintf("%s-%s", s.Name, s.Id)
}

// #region Options

type DefaultOptionConfigurable interface {
	SetHealthCheck(bool)
	SetTimeout(timeout time.Duration)
	SetDatacenter(dc string)
	SetHeartbeatEnabled(bool)
	SetHealthCheckInterval(int)
	SetDeregisterCriticalServiceAfter(int)
	SetTags(...string)
}

//#endregion
