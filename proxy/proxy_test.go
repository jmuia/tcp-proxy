package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/jmuia/tcp-proxy/loadbalancer"
	proxytesting "github.com/jmuia/tcp-proxy/testing"
	"github.com/pkg/errors"
)

// TODO: lifecycle tests
// - error on Accept shutsdown
// - integration of health checks, registry, load balancing

func TestProxy(t *testing.T) {
	// Set up a backend to proxy to.
	backendListener := proxytesting.NewLocalListener(t)
	defer backendListener.Close()

	// Set up proxy.
	tcpProxy := newSimpleTCPProxy(t, []string{backendListener.Addr().String()})

	err := tcpProxy.Start()
	defer tcpProxy.Shutdown()
	check(t, err)

	// Connect to the proxy as a client.
	client, err := net.Dial("tcp", tcpProxy.ln.Addr().String())
	defer client.Close()
	check(t, err)

	// Accept the connection from the proxy to the backend.
	backend, err := backendListener.Accept()
	defer backend.Close()
	check(t, err)

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
	check(t, err)
	backend.Close()
	client.Close()

	// Sleep for a couple seconds to allow the proxy to finish up
	// processing the above communications. :(
	time.Sleep(1 * time.Second)
}

func TestShutdownNoConnections(t *testing.T) {
	tcpProxy := newSimpleTCPProxy(t, []string{})

	err := tcpProxy.Start()
	check(t, err)

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

func TestCannotStartTwice(t *testing.T) {
	tcpProxy := newSimpleTCPProxy(t, []string{})

	err := tcpProxy.Start()
	defer tcpProxy.Shutdown()
	check(t, err)

	err = tcpProxy.Start()
	if err == nil {
		t.Error("expected proxy to error when attempting to start a second time")
	}
}

func TestIdempotentShutdown(t *testing.T) {
	tcpProxy := newSimpleTCPProxy(t, []string{})

	err := tcpProxy.Start()
	check(t, err)

	tcpProxy.Shutdown()

	// Calling shutdown a second time is ok.
	tcpProxy.Shutdown()
}

func newSimpleTCPProxy(t *testing.T, backends []string) *TCPProxy {
	proxyConfig := Config{
		Laddr:    "localhost:0",
		Timeout:  1 * time.Second,
		Backends: backends,
		Lb:       loadbalancer.Config{loadbalancer.P2C_TYPE},
	}
	tcpProxy, err := NewTCPProxy(proxyConfig)
	if err != nil {
		t.Fatal(err)
	}
	return tcpProxy
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

func TestClosesClientConnectionOnBackendError(t *testing.T) {
	// Set up proxy.
	tcpProxy := newSimpleTCPProxy(t, []string{})

	err := tcpProxy.Start()
	defer tcpProxy.Shutdown()
	check(t, err)

	// Connect to the proxy as a client -- but there are no healthy backends!
	client, err := net.Dial("tcp", tcpProxy.ln.Addr().String())
	defer client.Close()
	check(t, err)
}

func TestStats(t *testing.T) {
	// Set up a backend to proxy to.
	backendListener := proxytesting.NewLocalListener(t)
	defer backendListener.Close()

	// Set up proxy.
	tcpProxy := newSimpleTCPProxy(t, []string{backendListener.Addr().String()})

	err := tcpProxy.Start()
	defer tcpProxy.Shutdown()
	check(t, err)

	// Check active connections.
	stats := tcpProxy.Stats()
	backendMetricPrefix := "backend." + backendListener.Addr().String() + "."
	assertMetric(t, stats, backendMetricPrefix+"active_connections", uint64(0))

	// Connect to the proxy as a client.
	client, err := net.Dial("tcp", tcpProxy.ln.Addr().String())
	defer client.Close()
	check(t, err)

	// Accept the connection from the proxy to the backend.
	backend, err := backendListener.Accept()
	defer backend.Close()
	check(t, err)

	// Check stats.
	time.Sleep(1 * time.Millisecond)
	stats = tcpProxy.Stats()
	assertMetric(t, stats, backendMetricPrefix+"active_connections", uint64(1))
	assertMetric(t, stats, "requests", uint64(1))

	// Send data back and forth.
	check(t, assertSendAndReceiveMessage(client, backend, "hi!"))
	check(t, assertSendAndReceiveMessage(backend, client, "hello!"))
	backend.Close()
	client.Close()

	// Check stats.
	time.Sleep(1 * time.Millisecond)
	stats = tcpProxy.Stats()
	assertMetric(t, stats, backendMetricPrefix+"active_connections", uint64(0))
	assertMetric(t, stats, "frontend.io.rx", uint64(len("hi!")))
	assertMetric(t, stats, "frontend.io.tx", uint64(len("hello!")))
	assertMetric(t, stats, backendMetricPrefix+"io.tx", uint64(len("hi!")))
	assertMetric(t, stats, backendMetricPrefix+"io.rx", uint64(len("hello!")))

	// Connect to the proxy as a client -- but the backend is down!
	backendListener.Close()
	client, err = net.Dial("tcp", tcpProxy.ln.Addr().String())
	defer client.Close()
	check(t, err)

	// Check stats.
	time.Sleep(1 * time.Millisecond)
	stats = tcpProxy.Stats()
	assertMetric(t, stats, "requests", uint64(2))
	assertMetric(t, stats, "errors", uint64(1))
}

func assertMetric(t *testing.T, stats map[string]interface{}, name string, expected interface{}) {
	if stats[name] != expected {
		t.Errorf("expected %s to be %v, was %v: %v", name, expected, stats[name], stats)
	}
}

func check(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
