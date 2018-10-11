package main

import (
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
)

type ProxyConfig struct {
	laddr   string
	timeout time.Duration
}

type TCPProxy struct {
	laddr   string
	timeout time.Duration
}

func NewTCPProxy(cfg *ProxyConfig) *TCPProxy {
	return &TCPProxy{
		laddr:   cfg.laddr,
		timeout: cfg.timeout,
	}
}

func (t *TCPProxy) Start() error {
	logger.Info("starting proxy...")

	l, err := net.Listen("tcp", t.laddr)
	if err != nil {
		return errors.Wrapf(err, "failed to listen on %s", t.laddr)
	}
	defer l.Close()
	logger.Info("listening on ", t.laddr)

	for {
		src, err := l.Accept()
		if err != nil {
			logger.Error(errors.Wrapf(err, "error accepting connection"))
			continue
		}
		logger.Info("accepted connection from ", src.RemoteAddr())
		go t.handleConn(src)
	}
}

func (t *TCPProxy) handleConn(src net.Conn) {
	defer src.Close()

	service := "localhost:8000"

	dst, err := net.DialTimeout("tcp", service, t.timeout)
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
		laddr:   "localhost:8080",
		timeout: 5 * time.Second,
	}
	tcpProxy := NewTCPProxy(&proxyConfig)
	err := tcpProxy.Start()
	if err != nil {
		logger.Error(err)
	}
}
