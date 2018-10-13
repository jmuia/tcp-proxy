package main

import (
	"io"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type ProxyConfig struct {
	laddr    string
	timeout  time.Duration
	services []string
}

type TCPProxy struct {
	cfg       ProxyConfig
	ln        net.Listener
	state     ProxyState
	shutdownc chan struct{}
	exitc     chan error
}

func NewTCPProxy(cfg ProxyConfig) *TCPProxy {
	return &TCPProxy{
		cfg:       cfg,
		state:     READY,
		shutdownc: make(chan struct{}),
		exitc:     make(chan error, 1),
	}
}

func (t *TCPProxy) Shutdown() {
	logger.Info("shutting down...")
	prev := atomic.SwapUint32(&t.state, STOPPED)
	if prev != STOPPED {
		close(t.shutdownc)
	}
}

func (t *TCPProxy) Run() error {
	err := t.Start()
	if err != nil {
		return err
	}
	return <-t.exitc
}

func (t *TCPProxy) Start() error {
	logger.Info("starting proxy...")

	var err error
	t.ln, err = net.Listen("tcp", t.cfg.laddr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen on %s", t.cfg.laddr)
	}
	logger.Info("listening on ", t.ln.Addr())

	swapped := atomic.CompareAndSwapUint32(&t.state, READY, RUNNING)
	if !swapped {
		t.ln.Close()
		return errors.New("attempted to start proxy when not in READY state")
	}

	go t.acceptConns()
	return nil
}

func (t *TCPProxy) acceptConns() {
	defer t.ln.Close()
	defer close(t.exitc)

	for {
		select {
		case <-t.shutdownc:
			return
		default:
			src, err := t.ln.Accept()
			if err != nil {
				t.exitc <- err
				return
			}
			logger.Info("accepted connection from ", src.RemoteAddr())
			go t.handleConn(src)
		}
	}
}

func (t *TCPProxy) handleConn(src net.Conn) {
	defer src.Close()

	service := t.cfg.services[rand.Intn(len(t.cfg.services))]

	dst, err := net.DialTimeout("tcp", service, t.cfg.timeout)
	if err != nil {
		// TODO: attempt a different backend.
		logger.Error(errors.Wrapf(err, "error dialing service %s", service))
		return
	}
	defer dst.Close()
	logger.Info("opened connection to ", dst.RemoteAddr())

	t.proxyConn(src, dst)
}

func (t *TCPProxy) proxyConn(src net.Conn, dst net.Conn) {
	// Adding a buffer means that a goroutine can send a value
	// before there is a receiver.
	// We only receive one value on this channel, but the buffer
	// allows the second goroutine to send its value and exit.
	errc := make(chan error, 1)

	copy := func(dst net.Conn, src net.Conn) {
		bytes, err := io.Copy(dst, src)
		logger.Infof("proxied %v bytes from %v to %v", bytes, src.RemoteAddr(), dst.RemoteAddr())
		errc <- err
	}

	go copy(dst, src)
	go copy(src, dst)

	// Await an error or EOF from either goroutine.
	// The caller will close both connections, with the
	// consequence of causing the other (likely blocked)
	// goroutine to continue executing.
	err := <-errc
	if err != nil {
		logger.Error(errors.Wrapf(err, "error proxying data from %v to %v", src.RemoteAddr(), dst.RemoteAddr()))
	}
}

func main() {
	proxyConfig := ProxyConfig{
		laddr:    "localhost:8080",
		timeout:  5 * time.Second,
		services: []string{"localhost:8000"},
	}
	tcpProxy := NewTCPProxy(proxyConfig)
	err := tcpProxy.Run()
	if err != nil {
		logger.Error(err)
	}
}
