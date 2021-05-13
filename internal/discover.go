package internal

import (
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"time"
)

func Discover(ctx context.Context, n *Node, dht *dht.IpfsDHT, rendezvous string) {
	h := *n.Host
	routingDiscovery := discovery.NewRoutingDiscovery(dht)
	discovery.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := discovery.FindPeers(ctx, routingDiscovery, rendezvous)
			fmt.Printf("Network peer count: %d\n", len(peers))
			CheckError(err)

			for _, p := range peers {
				// Ignore self node
				if p.ID == h.ID() {
					continue
				}

				// Dial to the node if not connected
				if h.Network().Connectedness(p.ID) != network.Connected {
					_, err = h.Network().DialPeer(ctx, p.ID)
					fmt.Printf("Connecting to peer %s\n", p.ID.Pretty())
					if err != nil {
						fmt.Printf("Connect error: %+v\n", err.Error())
						continue
					}

				}
			}
		}
	}
}
