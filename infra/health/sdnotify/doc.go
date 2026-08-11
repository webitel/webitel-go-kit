// Package sdnotify reports the health registry to systemd over NOTIFY_SOCKET:
// READY=1 once ready, STATUS= on a state change, STOPPING=1 at shutdown, and
// WATCHDOG=1 while the registry's scheduler keeps turning.
//
// These variables are systemd's interface, so this package reads the
// environment where the rest of infra/health refuses to. No NOTIFY_SOCKET means
// [New] returns nil and every method is a no-op. WATCHDOG_USEC and WATCHDOG_PID
// are read once: the ping period is WATCHDOG_USEC/2, and a foreign WATCHDOG_PID
// or an unusable value disables the ping. A check's error text never reaches
// STATUS=, and a notify failure is logged rather than returned.
//
// systemd.go holds ~55 lines adapted from github.com/coreos/go-systemd/v22 under
// Apache 2.0, copied rather than imported to keep the require block empty; its
// header carries the attribution and the list of modifications.
//
// # For WTEL-10090
//
// WatchdogSec must be chosen against StaleAfter. The ping is gated on
// SchedulerAlive, so at period WATCHDOG_USEC/2 one skipped ping already makes
// the gap exactly WATCHDOG_USEC — the kill threshold. Use
// WatchdogSec >= 4 x StaleAfter (>= 60s at [health.DefaultConfig]), never below
// 2 x StaleAfter.
//
// Without [WithStartTimeout] there is no READY=1 fallback, so under Type=notify
// a unit whose critical check never goes green stays in activating until
// TimeoutStartSec. Type=notify, TimeoutStartSec=90 and WithStartTimeout belong
// in one commit.
package sdnotify
