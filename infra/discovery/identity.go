package discovery

import (
	"fmt"
	"os"
)

// GenerateInstanceID returns a stable, unique service instance identifier
// in the format: serviceName@hostname.
//
// Falls back to pid-<pid> if hostname resolution fails.
//
// In Kubernetes, os.Hostname() returns the pod name
// (e.g. "im-account-service-7d4b-xk2m"), so the result is globally unique.
// On a VM or bare-metal host, hostname is unique per machine.
//
// Example: "im-account-service@worker-01"
func GenerateInstanceID(serviceName string) string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("pid-%d", os.Getpid())
	}
	return fmt.Sprintf("%s@%s", serviceName, hostname)
}
