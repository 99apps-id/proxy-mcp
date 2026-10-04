// proxy-mcp rotates outbound HTTP(S) requests through a pool of proxy servers,
// so a client can reach APIs that block its own egress IP. It speaks MCP over
// stdio and needs no API key.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
)

func main() {
	config := flag.String("config", "", "path to proxy pool JSON config file")
	flag.Parse()
	if err := os.Setenv("PROXY_MCP_CONFIG", *config); err != nil {
		fmt.Fprintln(os.Stderr, "proxy-mcp:", err)
		os.Exit(1)
	}

	pool, err := newPoolFromConfig(*config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "proxy-mcp: config:", err)
		os.Exit(1)
	}
	go pool.healthLoop()

	server := &server{out: bufio.NewWriter(os.Stdout), pool: pool}
	if err := server.serve(os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, "proxy-mcp:", err)
		os.Exit(1)
	}
}
