package main

import (
	"context"
	"flag"
	"log"
	"sync"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/trans_network"
)

func main() {
	var wg sync.WaitGroup
	ctx := context.Background()

	config := &trans_network.TransNetworkConfig{}
	flag.Var(&config.BootstrapPeers, "peers", "Bootstrap Peers")
	flag.IntVar(&config.InternalPort, "p2p-port", 2145, "Internal Port Used by p2p network")
	flag.StringVar(&config.ForwardServiceAddr, "service-addr", ":8080", "Service addr used for service forward")
	flag.StringVar(&config.MgmtAddr, "mgmt-addr", "localhost:8082", "API base for management")
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

	select {}
}
