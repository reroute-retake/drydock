package ports

import (
	"fmt"
	"net"
	"testing"
)

func TestGatewayPortStableAndInRange(t *testing.T) {
	a := GatewayPort("payments")
	if a != GatewayPort("payments") {
		t.Fatal("gateway port must be deterministic for a space")
	}
	if a < gatewayBase || a >= gatewayBase+gatewaySpan {
		t.Fatalf("port %d out of range [%d,%d)", a, gatewayBase, gatewayBase+gatewaySpan)
	}
}

func TestFree(t *testing.T) {
	// Bind a port, then Free must report it as in use.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	if Free(port) {
		t.Fatalf("port %d is bound; Free should be false", port)
	}
	// Sanity: the address string we build is well-formed.
	if _, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port)); err != nil {
		t.Fatal(err)
	}
}
