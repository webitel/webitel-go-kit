// Package hostaddr selects host network addresses for service listeners.
package hostaddr

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
)

var (
	ErrNoPrivateIPv4       = errors.New("no private IPv4 address found")
	ErrMultiplePrivateIPv4 = errors.New("multiple private IPv4 addresses found")
)

// PrivateIPv4 returns the only private IPv4 address assigned to an up,
// running, non-loopback interface. It returns ErrNoPrivateIPv4 when there is
// none and ErrMultiplePrivateIPv4, listing the candidates, when there is more
// than one.
func PrivateIPv4() (netip.Addr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("list network interfaces: %w", err)
	}

	var found []netip.Addr
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagRunning == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ip, ok := toAddr(addr)
			if !ok || !ip.Is4() || !ip.IsPrivate() || slices.Contains(found, ip) {
				continue
			}
			found = append(found, ip)
		}
	}

	switch len(found) {
	case 0:
		return netip.Addr{}, ErrNoPrivateIPv4
	case 1:
		return found[0], nil
	default:
		list := make([]string, len(found))
		for i, ip := range found {
			list[i] = ip.String()
		}
		return netip.Addr{}, fmt.Errorf("%w: %s", ErrMultiplePrivateIPv4, strings.Join(list, ", "))
	}
}

// Advertise returns the address to advertise for a listener bound
// to listen. A specific IP is returned unchanged. An unspecified IP (0.0.0.0
// or ::) is replaced with PrivateIPv4, keeping the port.
func Advertise(listen netip.AddrPort) (netip.AddrPort, error) {
	if !listen.IsValid() {
		return netip.AddrPort{}, errors.New("invalid listen address")
	}
	if !listen.Addr().IsUnspecified() {
		return listen, nil
	}

	ip, err := PrivateIPv4()
	if err != nil {
		return netip.AddrPort{}, err
	}
	return netip.AddrPortFrom(ip, listen.Port()), nil
}

func toAddr(addr net.Addr) (netip.Addr, bool) {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	default:
		return netip.Addr{}, false
	}
	a, ok := netip.AddrFromSlice(ip)
	return a.Unmap(), ok
}
