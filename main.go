package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/jmuia/tcp-proxy/health"
	"github.com/jmuia/tcp-proxy/loadbalancer"
	logger "github.com/jmuia/tcp-proxy/logging"
	"github.com/jmuia/tcp-proxy/proxy"
)

func main() {
	cfg := cli()
	tcpProxy, err := proxy.NewTCPProxy(*cfg)
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	handleExitSignal(tcpProxy)
	handleStatsSignal(tcpProxy)

	err = tcpProxy.Run()
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}

func cli() *proxy.Config {
	cfg := proxy.Config{}

	flag.Usage = func() {
		fmt.Println("Usage: ./tcp-proxy [OPTIONS] <BACKEND>...")
		flag.PrintDefaults()
		fmt.Println()

		fmt.Println("Example:")
		fmt.Println("  ./tcp-proxy \\")
		fmt.Println("\t-laddr localhost:4000 \\")
		fmt.Println("\t-timeout 3s \\")
		fmt.Println("\t-lb random \\")
		fmt.Println("\tlocalhost:8001 \\")
		fmt.Println("\tlocalhost:8002")
	}

	flag.StringVar(&cfg.Laddr, "laddr", ":4000", "address to listen on")
	flag.DurationVar(&cfg.Timeout, "timeout", 3*time.Second, "backend dial timeout")

	flag.Var(newLbTypeVar(&cfg.Lb.Type, loadbalancer.P2C_TYPE), "lb", "load balancer algorithm (RANDOM|P2C)")

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cfg.Backends = flag.Args()

	cfg.Health = health.HealthCheckConfig{
		Timeout:            1 * time.Second,
		Interval:           5 * time.Second,
		UnhealthyThreshold: 3,
		HealthyThreshold:   3,
	}

	return &cfg
}

type lbTypeValue loadbalancer.Type

func newLbTypeVar(p *loadbalancer.Type, value loadbalancer.Type) *lbTypeValue {
	*p = value
	return (*lbTypeValue)(p)
}

func (v *lbTypeValue) String() string {
	return (*loadbalancer.Type)(v).String()
}

func (v *lbTypeValue) Set(s string) error {
	t, err := loadbalancer.ParseType(s)
	if err != nil {
		return err
	}
	*v = lbTypeValue(t)
	return nil
}

func sorted(m map[string]interface{}) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func handleExitSignal(tcpProxy *proxy.TCPProxy) {
	exitc := make(chan os.Signal)
	signal.Notify(exitc, exitSignals...)
	go func() {
		for range exitc {
			fmt.Println()
			tcpProxy.Shutdown()
		}
	}()
}

func handleStatsSignal(tcpProxy *proxy.TCPProxy) {
	statsc := make(chan os.Signal)
	signal.Notify(statsc, statsSignals...)
	go func() {
		for range statsc {
			fmt.Println()
			stats := tcpProxy.Stats()
			for _, k := range sorted(stats) {
				fmt.Printf("%s: %v\n", k, stats[k])
			}
		}
	}()
}
