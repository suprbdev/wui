// Package webserver provides the listener and URL helpers wui uses to
// serve WASM web builds. It is exported so apps built on wui (and the
// bundled cmd/serve tool) can reuse the same port-picking behavior.
package webserver

import (
	"fmt"
	"net"
)

// Port range scanned when Listen gets an empty address: an
// unprivileged, developer-conventional block starting at wui's
// default port.
const (
	AutoPortMin = 8765
	AutoPortMax = 8864
)

// Listen binds the given address, or — when addr is empty — picks a
// port automatically: loopback only, first free port in
// [AutoPortMin, AutoPortMax], falling back to an OS-assigned ephemeral
// port when the whole range is busy.
func Listen(addr string) (net.Listener, error) {
	if addr != "" {
		return net.Listen("tcp", addr)
	}
	for port := AutoPortMin; port <= AutoPortMax; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			return ln, nil
		}
	}
	return net.Listen("tcp", "localhost:0")
}

// URL derives a browser-openable URL from a bound listener address,
// mapping wildcard and loopback hosts to "localhost".
func URL(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "http://" + addr.String() + "/"
	}
	if ip := net.ParseIP(host); ip == nil || ip.IsUnspecified() || ip.IsLoopback() {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}
