package proxy

import (
	"io"
	"math/rand"
	"net"
	"time"

	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/jmuia/tcp-proxy/service"
	"github.com/pkg/errors"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type TCPProxy struct {
	cfg       ProxyConfig
	ln        net.Listener
	state     ProxyState
	registry  *service.ServiceRegistry
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

	// TODO: update load balancing based upon health checks.
	t.registry = service.NewServiceRegistry(t.cfg.Health)
	t.registry.RegisterUpdateListener(func(service service.Service) {
		logger.Infof("%s now %s", service.Addr(), service.State().String())
	})
	for _, s := range t.cfg.Services {
		err := t.registry.Add(s)
		if err != nil {
			t.Shutdown()
			return errors.Wrapf(err, "failed to register %s", s)
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

	services := t.registry.Snapshot()
	service := services[rand.Intn(len(services))]

	dst, err := net.DialTimeout("tcp", service.Addr(), t.cfg.Timeout)
	if err != nil {
		// TODO: attempt a different backend.
		logger.Error(errors.Wrapf(err, "error dialing service %v", service))
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
