package main

import (
	"flag"
	"log"

	"github.com/sequix/nbd/pkg/client"
)

var (
	network = flag.String("network", "tcp", "network, tcp or unix")
	addr    = flag.String("addr", "127.0.0.1:10809", "address")
	dev     = flag.String("dev", "/dev/nbd1", "nbd device path")
	export = flag.String("export", "export-1", "export to use")
)

func main() {
	flag.Parse()
	c, err := client.New(*network, *addr)
	if err != nil {
		log.Fatal(err)
	}
	if err := c.Go(*dev, *export); err != nil {
		log.Fatal(err)
	}
}
