// This file contains code adapted from github.com/coreos/go-systemd/v22 at
// v22.7.0, files daemon/sdnotify.go and daemon/watchdog.go. It is copied
// rather than imported so that this module keeps an empty require block; see
// infra/health/go.mod.
//
// Copyright 2014 Docker, Inc.
// Copyright 2015-2018 CoreOS, Inc.
// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// The rest of this repository is MIT; see the repository root LICENSE.
//
// NOTICE, reproduced from the upstream NOTICE file:
//
//	CoreOS Project
//	Copyright 2018 CoreOS, Inc
//
//	This product includes software developed at CoreOS, Inc.
//	(http://www.coreos.com/).
//
// MODIFICATIONS by Webitel, 2026 (Apache License 2.0, section 4(b)):
//
//   - SdNotify is unexported as notify and takes the socket address as a
//     parameter instead of reading NOTIFY_SOCKET on every call.
//   - notify sets a write deadline before writing. Upstream has none; that
//     gap is the reason this code is copied and modified rather than imported.
//   - notify rewrites a leading '@' to NUL for Linux abstract sockets.
//   - notify returns a plain error instead of (bool, error). New already
//     handles the "NOTIFY_SOCKET is unset" case, so the bool carried no
//     information at this call site.
//   - The unsetEnvironment parameter is removed from both functions; this
//     package never mutates the process environment.
//   - SdWatchdogEnabled is unexported as watchdogEnabled, and range-checks
//     WATCHDOG_USEC against maxWatchdogUsec. Upstream's strconv.Atoi lets a
//     large value overflow the multiplication into a spin-loop period.
//   - SdNotifyReloading, SdNotifyMonotonicUsec and the sdnotify_unix.go /
//     sdnotify_other.go build-tagged pair are not copied. The first is unused
//     here; the second needs golang.org/x/sys/unix, which this module must
//     not depend on.
//   - The SdNotify* constants are unexported and renamed to notifyReady,
//     notifyStopping and notifyWatchdog.
//   - notify wraps its socket errors with %w and a "health/sdnotify:" prefix.
//     Upstream returns the bare *net.OpError.
//   - The "NOTIFY_SOCKET is unset" early return is deleted. New owns that case
//     and returns a nil *Notifier, so notify is never reached with an empty
//     address.

package sdnotify

import (
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	notifyReady    = "READY=1"
	notifyStopping = "STOPPING=1"
	notifyWatchdog = "WATCHDOG=1"
)

// maxWatchdogUsec is the largest WATCHDOG_USEC that survives the conversion to
// a Duration. Beyond it the multiplication wraps, and a wrapped value lands
// wherever it lands — a sub-microsecond period turns the loop into a spin loop
// that floods the notify socket, which is worse than no watchdog at all.
const maxWatchdogUsec = int64(math.MaxInt64) / int64(time.Microsecond)

// notify sends one datagram to the systemd notify socket.
func notify(addr, state string, timeout time.Duration) error {
	// Belt and braces: Go's linux syscall layer accepts '@' and '\x00'
	// identically (syscall/syscall_linux.go:547-577), so this is a no-op there.
	// Abstract sockets do not exist off linux, where dialling an '@' address
	// fails either way.
	if strings.HasPrefix(addr, "@") {
		addr = "\x00" + addr[1:]
	}

	socketAddr := &net.UnixAddr{
		Name: addr,
		Net:  "unixgram",
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		return fmt.Errorf("health/sdnotify: dial %s: %w", addr, err)
	}
	defer conn.Close()

	_ = conn.SetWriteDeadline(time.Now().Add(timeout))

	if _, err = conn.Write([]byte(state)); err != nil {
		return fmt.Errorf("health/sdnotify: write %s: %w", addr, err)
	}

	return nil
}

// watchdogEnabled returns the full WATCHDOG_USEC interval, or 0 when the
// watchdog is not enabled for this process. The caller halves it.
func watchdogEnabled() (time.Duration, error) {
	wusec := os.Getenv("WATCHDOG_USEC")
	wpid := os.Getenv("WATCHDOG_PID")

	if wusec == "" {
		return 0, nil
	}
	s, err := strconv.ParseInt(wusec, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting WATCHDOG_USEC: %w", err)
	}
	if s <= 0 || s > maxWatchdogUsec {
		return 0, fmt.Errorf("error WATCHDOG_USEC out of range: %s", wusec)
	}
	interval := time.Duration(s) * time.Microsecond

	if wpid == "" {
		return interval, nil
	}
	p, err := strconv.Atoi(wpid)
	if err != nil {
		return 0, fmt.Errorf("error converting WATCHDOG_PID: %w", err)
	}
	if os.Getpid() != p {
		return 0, nil
	}

	return interval, nil
}
