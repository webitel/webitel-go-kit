// Package healthhttp serves the health registry over HTTP — /livez, /readyz and
// /healthz — as a handler to mount, or on its own listener.
//
// The package name differs from its directory so it does not shadow net/http at
// import sites. Routing is on the last path segment, so one handler works
// mounted at the root, under a prefix, or at one exact path; [LivenessHandler],
// [ReadinessHandler] and [HealthHandler] ignore the path entirely.
//
// A check's error text never reaches a response body. ?verbose upgrades /livez
// and /readyz to the JSON /healthz already serves, and only on a [Server] bound
// to loopback.
package healthhttp
