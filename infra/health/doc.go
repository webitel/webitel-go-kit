// Package health runs service health checks in the background and caches the
// last result. Transports read the cache and never run a check themselves, so
// the number of readers has no effect on the load a dependency sees.
//
//   - health/http — /livez, /readyz, /healthz
//   - health/sdnotify — systemd READY, STATUS, STOPPING, WATCHDOG
//
// Service discovery takes the verdict from [Registry.ReadyFunc]. The package
// reads no environment, files or flags; callers fill [Config] themselves, or
// take [DefaultConfig]. Standard library only.
package health
