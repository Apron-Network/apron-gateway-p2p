package trans_network

import (
	"context"
	"log"
	"sync"

	"apron.network/gateway-p2p/internal"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func NewKDHT(ctx context.Context, host host.Host, bootstrapPeers []multiaddr.Multiaddr, wg *sync.WaitGroup) (*dht.IpfsDHT, error) {
	var options []dht.Option

	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	kdht, err := dht.New(ctx, host, options...)
	internal.CheckError(err)

	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	// Connect to bootstrap nodes
	for _, peerAddr := range bootstrapPeers {
		peerInfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerInfo); err != nil {
				log.Printf("Error while connecting to node %q: %-v", peerInfo, err)
			} else {
				log.Printf("Connection established with bootstrap node: %q", *peerInfo)
			}
		}()
	}
	wg.Wait()

	return kdht, nil
}
