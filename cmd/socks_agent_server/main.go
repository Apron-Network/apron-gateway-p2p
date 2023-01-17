package main

import (
	"flag"
	"log"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/ext_protocols/socks5"
	"go.uber.org/zap"
)

func startApronSocks5ServiceSideServer(listenAddr string) {
	logger, err := zap.NewProduction()
	internal.CheckError(err)

	// Socks5 service
	conf := &socks5.Config{
		Logger: logger,
	}

	server, err := socks5.NewApronServer(conf, socks5.ServerSideMode)
	if err != nil {
		logger.Error("new socks5 server error", zap.Error(err))
		panic(err)
	}

	if err := server.ListenAndServe("tcp", listenAddr); err != nil {
		logger.Error("listen and serve error", zap.Error(err))
		panic(err)
	}
}

func main() {
	listenAddr := flag.String("listenAddr", ":10800", "listen address for server")
	flag.Parse()

	log.Printf("Starting service at %s ...\n", *listenAddr)
	go startApronSocks5ServiceSideServer(*listenAddr)

	select {}
}
