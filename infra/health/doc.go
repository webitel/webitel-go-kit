// Package health runs service health checks in the background and caches the
// last result. Transports read the cache and never run a check themselves.
package health
