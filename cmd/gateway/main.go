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

	config := &trans_network.TransNetworkConfig{}
	flag.Var(&config.BootstrapPeers, "peers", "Bootstrap Peers")
	flag.IntVar(&config.ConnectPort, "port", 2145, "Service port")
	flag.Parse()

	node, err := trans_network.NewNode(ctx, config)
	internal.CheckError(err)

	log.Printf("Host ID: %s", (*node.Host).ID().Pretty())
	log.Printf("Connect to me on:")
	for _, addr := range (*node.Host).Addrs() {
		log.Printf("  %s/p2p/%s", addr, (*node.Host).ID().Pretty())
	}

	kdht, err := trans_network.NewKDHT(ctx, *node.Host, config.BootstrapPeers, &wg)
	internal.CheckError(err)

	// Setup listener for service broadcast
	node.SetupServiceBroadcastListener(ctx)

	// Start discover goroutines
	go trans_network.Discover(ctx, node, kdht, "asdfasdf")

	// TODO: Start management APIs

	select {}
}
