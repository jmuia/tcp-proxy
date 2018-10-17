package testing

import (
	"net"
	"testing"
)

func NewLocalListener(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		return ln
	}
	ln, err = net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}
