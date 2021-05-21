package test

import (
	"apron.network/gateway-p2p/internal/trans_network"
	"context"
	"fmt"
	"sync"
)

func BuildKdhtNetwork(ctx context.Context, bsNodeCount int, clientCount int) ([]*trans_network.Node, []*trans_network.Node) {
	bsNodes := make([]*trans_network.Node, bsNodeCount)
	clientNodes := make([]*trans_network.Node, clientCount)
	bsPeers := trans_network.AddrList{}

	wg := sync.WaitGroup{}
	rendezvous := "foobar"

	startPort := 12345
	port := startPort

	// Build bootstrap nodes
	for i := 0; i < bsNodeCount; i++ {
		config := &trans_network.TransNetworkConfig{
			BootstrapPeers: nil,
			ConnectPort:    port,
		}

		node, err := trans_network.NewNode(ctx, config)
		if err != nil {
			panic(err)
		}

		node.SetupServiceBroadcastListener(ctx)
		bsNodes[i] = node

		bsIdStr := (*node.Host).ID().Pretty()
		bsNodeAddr := (*node.Host).Addrs()[0]
		bsNodeAddrStr := fmt.Sprintf("%s/p2p/%s", bsNodeAddr, bsIdStr)

		bsPeers.Set(bsNodeAddrStr)
		port++

		kdht, err := trans_network.NewKDHT(ctx, *node.Host, nil, &wg)
		if err != nil {
			panic(err)
		}
		go trans_network.Discover(ctx, node, kdht, rendezvous)
	}

	// Build client nodes
	for i := 0; i < clientCount; i++ {
		fmt.Printf("bootstrap peers: %+v\n", bsPeers)
		config := &trans_network.TransNetworkConfig{
			BootstrapPeers: bsPeers,
			ConnectPort:    port,
		}

		node, err := trans_network.NewNode(ctx, config)
		if err != nil {
			panic(err)
		}

		node.SetupServiceBroadcastListener(ctx)
		clientNodes[i] = node
		port++

		kdht, err := trans_network.NewKDHT(ctx, *node.Host, bsPeers, &wg)
		if err != nil {
			panic(err)
		}
		go trans_network.Discover(ctx, node, kdht, rendezvous)
	}

	return bsNodes, clientNodes
}
