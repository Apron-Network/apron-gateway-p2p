package trans_network

import (
	"context"
	"time"

	"apron.network/gateway-p2p/internal"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

func Discover(ctx context.Context, n *Node, dht *dht.IpfsDHT, rendezvous string) {
	h := *n.Host
	routingDiscovery := drouting.NewRoutingDiscovery(dht)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers, err := dutil.FindPeers(ctx, routingDiscovery, rendezvous)
			// fmt.Printf("%s: Network peer count: %d\n", (*n.Host).ID().Pretty(), len(peers))
			internal.CheckError(err)

			for _, p := range peers {
				// Ignore self node
				if p.ID == h.ID() {
					continue
				}

				// Dial to the node if not connected
				// if h.Network().Connectedness(p.ID) != network.Connected {
				// 	_, err = h.Network().DialPeer(ctx, p.ID)
				// 	fmt.Printf("Connecting to peer %s\n", p.ID.Pretty())
				// 	if err != nil {
				// 		fmt.Printf("Connect error: %+v\n", err.Error())
				// 		continue
				// 	}

				// }
			}
		}
	}
}
