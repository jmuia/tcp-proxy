package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	cfg := cli()
	tcpProxy := NewTCPProxy(*cfg)
	err := tcpProxy.Run()
	if err != nil {
		logger.Error(err)
	}
}

func cli() *ProxyConfig {
	cfg := ProxyConfig{}

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

	flag.StringVar(&cfg.laddr, "laddr", ":4000", "address to listen on")
	flag.DurationVar(&cfg.timeout, "timeout", 3*time.Second, "service dial timeout")

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	cfg.services = flag.Args()

	return &cfg
}
