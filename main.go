package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/jmuia/tcp-proxy/proxy"
)

func main() {
	cfg := cli()
	tcpProxy := proxy.NewTCPProxy(*cfg)

	signalc := make(chan os.Signal)
	signal.Notify(signalc, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range signalc {
			fmt.Println()
			tcpProxy.Shutdown()
		}
	}()

	err := tcpProxy.Run()
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func cli() *proxy.ProxyConfig {
	cfg := proxy.ProxyConfig{}

	flag.Usage = func() {
		fmt.Println("Usage: ./tcp-proxy [OPTIONS] <SERVICE>...")
		flag.PrintDefaults()
		fmt.Println()

		fmt.Println("Example:")
		fmt.Println("  ./tcp-proxy \\")
		fmt.Println("\t-laddr localhost:4000 \\")
		fmt.Println("\t-timeout 3s \\")
		fmt.Println("\tlocalhost:8001 \\")
		fmt.Println("\tlocalhost:8002")
	}

	flag.StringVar(&cfg.Laddr, "laddr", ":4000", "address to listen on")
	flag.DurationVar(&cfg.Timeout, "timeout", 3*time.Second, "service dial timeout")

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cfg.Services = flag.Args()

	cfg.Health = health.HealthCheckConfig{
		Timeout:            1 * time.Second,
		Interval:           5 * time.Second,
		UnhealthyThreshold: 3,
		HealthyThreshold:   3,
	}

	return &cfg
}
