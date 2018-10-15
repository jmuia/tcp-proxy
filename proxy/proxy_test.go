package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/pkg/errors"
)

// TODO: lifecycle tests
// - cannot start twice
// - can shutdown idempotently
// - error on Accept shutsdown
// - shutdown without any running connections works

// - health check tests

func newLocalListener(t *testing.T) net.Listener {
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

func TestProxy(t *testing.T) {
	// Set up a service to proxy to.
	serviceListener := newLocalListener(t)
	defer serviceListener.Close()

	// Set up proxy.
	proxyConfig := Config{
		Laddr:    "localhost:0",
		Timeout:  1 * time.Second,
		Services: []string{serviceListener.Addr().String()},
		Health: health.HealthCheckConfig{
			Timeout:            1 * time.Second,
			Interval:           5 * time.Second,
			UnhealthyThreshold: 3,
			HealthyThreshold:   3,
		},
	}
	tcpProxy := NewTCPProxy(proxyConfig)
	err := tcpProxy.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer tcpProxy.Shutdown()

	// Connect to the proxy as a client.
	client, err := net.Dial("tcp", tcpProxy.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Accept the connection from the proxy to the service.
	service, err := serviceListener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()

	// Concurrently send messages back and forth,
	// tracking the first error that occurs.
	var wg sync.WaitGroup
	errc := make(chan error, 1)

	communicateViaProxy := func(s net.Conn, r net.Conn, msg string) {
		err := assertSendAndReceiveMessage(s, r, msg)
		if err != nil {
			// Don't block on sending an error if one
			// already exists.
			select {
			case errc <- err:
			default:
			}
		}
		wg.Done()
	}

	for i := 0; i < 1000; i++ {
		go communicateViaProxy(client, service, "hello!")
		go communicateViaProxy(service, client, "hey!")
		wg.Add(2)
	}
	wg.Wait()
	close(errc)
	err = <-errc
	if err != nil {
		t.Fatal(err)
	}
	service.Close()
	client.Close()

	// Sleep for a couple seconds to allow the proxy to finish up
	// processing the above communications. :(
	time.Sleep(1 * time.Second)
}

func assertSendAndReceiveMessage(s net.Conn, r net.Conn, msg string) error {
	_, err := io.WriteString(s, msg)
	if err != nil {
		return err
	}

	buf := make([]byte, len(msg))
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return err
	}

	if string(buf) != msg {
		errMsg := fmt.Sprintf("expected %q != actual %q", msg, buf)
		return errors.New(errMsg)
	}

	return nil
}
