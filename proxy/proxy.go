package proxy

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/jmuia/tcp-proxy/loadbalancer"
	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/jmuia/tcp-proxy/backend"
	"github.com/pkg/errors"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type TCPProxy struct {
	cfg       Config
	ln        net.Listener
	state     State
	lb        loadbalancer.LoadBalancer
	registry  *backend.Registry
	shutdownc chan struct{}
	exitc     chan error
}

func NewTCPProxy(cfg Config) (*TCPProxy, error) {
	var lb loadbalancer.LoadBalancer
	switch cfg.Lb.Type {
	case loadbalancer.RANDOM_TYPE:
		lb = loadbalancer.NewRandom()
	case loadbalancer.P2C_TYPE:
		lb = loadbalancer.NewP2C()
	default:
		return nil, errors.New(fmt.Sprintf("unexpected load balancer type %s", cfg.Lb.Type))
	}

	return &TCPProxy{
		cfg:       cfg,
		state:     NEW,
		lb:        lb,
		shutdownc: make(chan struct{}),
		exitc:     make(chan error, 1),
	}, nil
}

func (t *TCPProxy) Start() error {
	logger.Info("starting proxy...")
	logger.Infof("config: %+v", t.cfg)
	swapped := AtomicCompareAndSwap(&t.state, NEW, STARTING)
	if !swapped {
		return errors.New("attempted to start proxy when not in NEW state")
	}

	var err error
	t.ln, err = net.Listen("tcp", t.cfg.Laddr)
	if err != nil {
		t.Shutdown()
		return errors.Wrapf(err, "failed to listen on %s", t.cfg.Laddr)
	}

	logger.Info("listening on ", t.ln.Addr())

	t.registry = backend.NewRegistry(t.cfg.Health)
	t.registry.RegisterUpdateListener(func(backend *backend.Backend) {
		logger.Infof("%s now %s", backend.Addr(), backend.State().String())
	})
	t.registry.RegisterUpdateListener(func(backend *backend.Backend) {
		t.lb.UpdateBackend(backend)
	})
	for _, b := range t.cfg.Backends {
		err := t.registry.Add(b)
		if err != nil {
			t.Shutdown()
			return errors.Wrapf(err, "failed to register %s", b)
		}
	}

	swapped = AtomicCompareAndSwap(&t.state, STARTING, RUNNING)
	if !swapped {
		t.Shutdown()
		return errors.New("attempted to run proxy when not in STARTING state")
	}

	go t.acceptConns()
	return nil
}

func (t *TCPProxy) Shutdown() {
	prev := AtomicSwap(&t.state, STOPPED)
	logger.Infof("shutting down in state %s", prev.String())
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
	t.registry.EvictAll()
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
	// TODO: don't accept connections if there aren't any
	//       healthy backends to proxy to.
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

	backend := t.lb.NextBackend(src)

	dst, err := net.DialTimeout("tcp", backend.Addr(), t.cfg.Timeout)
	if err != nil {
		// TODO: attempt a different backend.
		logger.Error(errors.Wrapf(err, "error dialing backend %v", backend))
		return
	}
	defer dst.Close()

	activeConns := backend.IncrActiveConns()
	defer backend.DecrActiveConns()

	logger.Infof("opened connection to %s (%d active)", dst.RemoteAddr(), activeConns)

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
