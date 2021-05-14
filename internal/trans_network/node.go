package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

type Node struct {
	Host *host.Host

	ps     *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	selfID peer.ID
}

func NewNode(ctx context.Context, config *TransNetworkConfig) (*Node, error) {
	r := rand.Reader
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.ConnectPort))
	h, err := libp2p.New(ctx, libp2p.ListenAddrs(addr), libp2p.Identity(priv))
	if err != nil {
		return nil, err
	}

	return &Node{Host: &h}, nil
}

func (n *Node) SetupListener(ctx context.Context) {
	var err error
	n.ps, err = pubsub.NewGossipSub(ctx, *n.Host)
	internal.CheckError(err)

	n.topic, err = n.ps.Join(BroadcastLocalServiceTopic)
	internal.CheckError(err)

	n.sub, err = n.topic.Subscribe()
	internal.CheckError(err)

	n.selfID = (*n.Host).ID()

	go n.StartListening(ctx)
}

func (n *Node) StartListening(ctx context.Context) {
	for {
		msg, err := n.sub.Next(ctx)
		internal.CheckError(err)

		if msg.ReceivedFrom == n.selfID {
			continue
		}

		fmt.Printf("Received msg: %+v\n", msg)
	}
}

func (n *Node) Publish(ctx context.Context, msg string) error {
	return n.topic.Publish(ctx, []byte(msg))
}
