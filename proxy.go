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
		state:     NEW,
		shutdownc: make(chan struct{}),
		exitc:     make(chan error, 1),
	}
}

func (t *TCPProxy) Start() error {
	logger.Info("starting proxy...")
	swapped := atomic.CompareAndSwapUint32(&t.state, NEW, STARTING)
	if !swapped {
		return errors.New("attempted to start proxy when not in NEW state")
	}

	var err error
	t.ln, err = net.Listen("tcp", t.cfg.laddr)
	if err != nil {
		t.Shutdown()
		return errors.Wrapf(err, "failed to listen on %s", t.cfg.laddr)
	}

	logger.Info("listening on ", t.ln.Addr())

	swapped = atomic.CompareAndSwapUint32(&t.state, STARTING, RUNNING)
	if !swapped {
		t.Shutdown()
		return errors.New("attempted to run proxy when not in STARTING state")
	}

	go t.acceptConns()
	return nil
}

func (t *TCPProxy) Shutdown() {
	prev := atomic.SwapUint32(&t.state, STOPPED)
	logger.Infof("shutting down in state %s", ProxyStateString(prev))
	switch prev {
	case NEW, STARTING:
		close(t.shutdownc)
		t.exit()
	case RUNNING:
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

func (t *TCPProxy) exit() {
	if t.ln != nil {
		t.ln.Close()
	}
	close(t.exitc)
}

func (t *TCPProxy) acceptTimeout(timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	err := t.ln.(*net.TCPListener).SetDeadline(deadline)
	if err != nil {
		return nil, err
	}
	return t.ln.Accept()
}

func (t *TCPProxy) acceptConns() {
	defer t.exit()

	// TODO: use a worker pool to limit concurrency.
	for {
		select {
		case <-t.shutdownc:
			return
		default:
			// Accept() is blocking. Adds a timeout
			// to ensure we're still checking for
			// shutdown messages if the proxy is idle.
			src, err := t.acceptTimeout(3 * time.Second)
			if isTimeout(err) {
				continue
			}
			if err != nil {
				t.exitc <- err
				t.Shutdown()
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

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	opErr, ok := err.(*net.OpError)
	return ok && opErr.Timeout()
}
