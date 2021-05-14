package main

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/trans_network"
	"context"
	"flag"
	"log"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	ctx := context.Background()

	// TODO: Parse CLI params to config object
	config := &internal.GatewayConfig{}
	peerList := trans_network.AddrList{}

	flag.Var(&peerList, "peers", "Bootstrap Peers")
	port := flag.Int("port", 2145, "Service port")
	flag.Parse()

	node, err := trans_network.NewNode(ctx, config)
	internal.CheckError(err)

	log.Printf("Host ID: %s", (*node.Host).ID().Pretty())
	log.Printf("Connect to me on:")
	for _, addr := range (*node.Host).Addrs() {
		log.Printf("  %s/p2p/%s", addr, (*node.Host).ID().Pretty())
	}

	kdht, err := trans_network.NewKDHT(ctx, *node.Host, peerList, &wg)
	internal.CheckError(err)

	// Setup pubsub
	node.SetupListener(ctx)

	// Start discover goroutines
	go trans_network.Discover(ctx, node, kdht, "asdfasdf")

	// TODO: Start management APIs

	select {}
}
