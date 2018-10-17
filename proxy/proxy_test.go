package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/jmuia/tcp-proxy/loadbalancer"
	proxytesting "github.com/jmuia/tcp-proxy/testing"
	"github.com/pkg/errors"
)

// TODO: lifecycle tests
// - cannot start twice
// - can shutdown idempotently
// - error on Accept shutsdown

// - health check tests

func TestProxy(t *testing.T) {
	// Set up a backend to proxy to.
	backendListener := proxytesting.NewLocalListener(t)
	defer backendListener.Close()

	// Set up proxy.
	proxyConfig := Config{
		Laddr:    "localhost:0",
		Timeout:  1 * time.Second,
		Backends: []string{backendListener.Addr().String()},
		Health: health.HealthCheckConfig{
			Timeout:            1 * time.Second,
			Interval:           5 * time.Second,
			UnhealthyThreshold: 3,
			HealthyThreshold:   3,
		},
		Lb: loadbalancer.Config{loadbalancer.P2C_TYPE},
	}

	tcpProxy, err := NewTCPProxy(proxyConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = tcpProxy.Start()
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

	// Accept the connection from the proxy to the backend.
	backend, err := backendListener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()

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
		go communicateViaProxy(client, backend, "hello!")
		go communicateViaProxy(backend, client, "hey!")
		wg.Add(2)
	}
	wg.Wait()
	close(errc)
	err = <-errc
	if err != nil {
		t.Fatal(err)
	}
	backend.Close()
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

func TestShutdownNoConnections(t *testing.T) {
	proxyConfig := Config{
		Laddr:   "localhost:0",
		Timeout: 1 * time.Second,
		Lb:      loadbalancer.Config{loadbalancer.P2C_TYPE},
	}

	tcpProxy, err := NewTCPProxy(proxyConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = tcpProxy.Start()
	if err != nil {
		t.Fatal(err)
	}

	donec := make(chan struct{})
	go func() {
		tcpProxy.Shutdown()
		close(donec)
	}()

	select {
	case <-donec:
	case <-time.NewTimer(5 * time.Second).C:
		t.Error("proxy didn't shutdown in 5s")
	}
}
