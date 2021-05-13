package main

import (
	"apron.network/gateway-p2p/internal"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"

	"io"
	"log"
	"sync"
)



var wg sync.WaitGroup

func NewHost(ctx context.Context, seed int64, port int) (host.Host, error) {
	var r io.Reader
	r = rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	internal.CheckError(err)

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	return libp2p.New(ctx, libp2p.ListenAddrs(addr), libp2p.Identity(priv))
}

func NewKDHT(ctx context.Context, host host.Host, bootstrapPeers []multiaddr.Multiaddr) (*dht.IpfsDHT, error) {
	var options []dht.Option

	if len(bootstrapPeers) == 0 {
		options = append(options, dht.Mode(dht.ModeServer))
	}

	kdht, err := dht.New(ctx, host, options...)
	internal.CheckError(err)

	if err = kdht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				log.Printf("Error while connecting to node %q: %-v", peerinfo, err)
			} else {
				log.Printf("Connection established with bootstrap node: %q", *peerinfo)
			}
		}()
	}
	wg.Wait()

	return kdht, nil
}

func main() {
	ctx := context.Background()

	peerList := internal.AddrList{}

	flag.Var(&peerList, "peer", "Bootstrap peers")
	serviceName := flag.String("serviceName", "services", "Services name")
	port := flag.Int("port", 2145, "Service port")
	flag.Parse()

	node, err := internal.NewNode(ctx, *port)
	internal.CheckError(err)

	log.Printf("Host ID: %s", (*node.Host).ID().Pretty())
	log.Printf("Connect to me on:")
	for _, addr := range (*node.Host).Addrs() {
		log.Printf("  %s/p2p/%s", addr, (*node.Host).ID().Pretty())
	}

	kdht, err := NewKDHT(ctx, *node.Host, peerList)
	internal.CheckError(err)

	selfServices := []string{*serviceName}

	fmt.Printf("Self services: %+v\n", selfServices)

	//// Method 1: Setup RPC
	//service := &UpdateServiceRpcCall{
	//	host:       h,
	//	protocolID: protocol.ID("UpdateServiceRpcProtocol"),
	//}
	//err = service.SetupRPC()
	//internal.CheckError(err)
	//go service.BroadcastLocalService(&selfServices)

	// Method 2: Setup pubsub
	node.SetupListener(ctx)

	go internal.Discover(ctx, node, kdht, "asdfasdf")

	select {}
}
