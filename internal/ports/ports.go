// Package ports handles host-port allocation for concurrent spaces. Each space
// gets a deterministic gateway host port so two running spaces don't both try
// to bind 4000, and callers can check whether a host port is free before start.
package ports

import (
	"fmt"
	"hash/fnv"
	"net"
)

const (
	gatewayBase = 41000
	gatewaySpan = 1000 // gateway host ports live in [41000, 41999]
)

// GatewayPort returns a stable host port for a space's LiteLLM gateway, derived
// from the space name. (The dev container always reaches the gateway internally
// at gateway:4000; this is only the host-side mapping, so it must be unique per
// concurrently-running space.)
func GatewayPort(space string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(space))
	return gatewayBase + int(h.Sum32()%gatewaySpan)
}

// Free reports whether a TCP host port can currently be bound.
func Free(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
