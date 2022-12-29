package main

import (
	"flag"
	"log"

	"apron.network/gateway-p2p/internal/ext_protocols/socks5"
)

func main() {
	listenAddr := flag.String("listenAddr", ":10800", "listen address for server")
	flag.Parse()

	// Socks5 service
	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	log.Printf("Starting service at %s ...", *listenAddr)
	if err := server.ListenAndServe("tcp", *listenAddr, false); err != nil {
		panic(err)
	}
}
