package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"time"
)

var HttpMethodMapping = map[models.ApronServiceRequest_HttpMethod]string{
	models.ApronServiceRequest_GET:    "GET",
	models.ApronServiceRequest_PUT:    "PUT",
	models.ApronServiceRequest_POST:   "POST",
	models.ApronServiceRequest_DELETE: "DELETE",
}

type Node struct {
	Host *host.Host

	ps                    *pubsub.PubSub
	broadcastServiceTopic *pubsub.Topic
	serviceBroadcastSub   *pubsub.Subscription
	selfID                peer.ID

	localServices []models.ApronService
	remoteService map[string][]models.ApronService
}

func NewNode(ctx context.Context, config *TransNetworkConfig) (*Node, error) {
	// r := rand.Reader
	// priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	// if err != nil {
	//	return nil, err
	// }
	//
	// addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.ConnectPort))
	// h, err := libp2p.New(ctx, libp2p.ListenAddrs(addr), libp2p.Identity(priv))
	h, err := libp2p.New(context.Background(), libp2p.ListenAddrs())
	if err != nil {
		return nil, err
	}

	return &Node{Host: &h}, nil
}

// SetupServiceBroadcastListener set subscriber of service broadcast
func (n *Node) SetupServiceBroadcastListener(ctx context.Context) {
	var err error
	n.ps, err = pubsub.NewGossipSub(ctx, *n.Host)
	internal.CheckError(err)

	n.broadcastServiceTopic, err = n.ps.Join(BroadcastServiceChannel)
	internal.CheckError(err)

	n.serviceBroadcastSub, err = n.broadcastServiceTopic.Subscribe()
	internal.CheckError(err)

	n.selfID = (*n.Host).ID()

	go n.StartListeningOnServiceBroadcast(ctx)
}

// StartListeningOnServiceBroadcast is a infinity loop which listens to service broadcast subscriptions.
// After receiving the message, the service will be added to remote service list of this node,
// which will be queried while receiving service request
func (n *Node) StartListeningOnServiceBroadcast(ctx context.Context) {
	for {
		msg, err := n.serviceBroadcastSub.Next(ctx)
		internal.CheckError(err)

		if msg.ReceivedFrom == n.selfID {
			continue
		}

		fmt.Printf("Received msg: %+v\n", msg)
	}
}

// BroadcastService broad local service to the network with configured topic,
// so all nodes subscribed to the topic can update its local cache data
func (n *Node) BroadcastService(ctx context.Context, msg string) error {
	return n.broadcastServiceTopic.Publish(ctx, []byte(msg))
}

func (n *Node) RegisterService(service *models.ApronService) {
	n.localServices = append(n.localServices, *service)
}

func (n *Node) NodeAddrStr() string {
	bsIdStr := (*n.Host).ID().Pretty()
	bsNodeAddr := (*n.Host).Addrs()[0]
	return fmt.Sprintf("%s/p2p/%s", bsNodeAddr, bsIdStr)
}

// TODO: when new node join the network, it require to connect to all gw, and get services registered on the node

// NewProxyRequest send proxy request to node associated with service.
// The request is encapsulated in a proto message, and after final message sent, a \n will be sent to note message done,
// this should be changed to some other policy in future.
func (n *Node) NewProxyRequest(remoteNode *Node, proxyReq *models.ApronServiceRequest) {
	s, err := (*n.Host).NewStream(context.Background(), (*remoteNode.Host).ID(), protocol.ID(ProxyRequestStream))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Stream: %+v\n", s)

	// Build request package with passed in request
	reqBytes, err := proto.Marshal(proxyReq)
	internal.CheckError(err)
	err = WriteBytesViaStream(s, reqBytes)
	internal.CheckError(err)

	// Read response from remote
	respBytes, err := ReadBytesViaStream(s)
	internal.CheckError(err)

	if err != nil {
		panic(err)
	}
	fmt.Printf("Resp: %s\n", string(respBytes))
}

// SetProxyRequestStreamHandler set handler to process proxy request
func (n *Node) SetProxyRequestStreamHandler() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyRequestStream), func(s network.Stream) {
		// TODO: Forward request to remote service, and send respond to invoker via respstream
		// Q: How to get src addr

		msgByte, err := ReadBytesViaStream(s)
		internal.CheckError(err)

		proxyReq := &models.ApronServiceRequest{}
		err = proto.Unmarshal(msgByte, proxyReq)

		fmt.Printf("Read stream: %s\n", proxyReq)

		// Build http request and send
		serviceUrl := fmt.Sprintf("%s://%s", proxyReq.Schema, proxyReq.ServiceUrl)
		fmt.Printf("%s: %s\n", models.ApronServiceRequest_HttpMethod_name[int32(proxyReq.HttpMethod)], serviceUrl)

		netClient := &http.Client{
			Timeout: time.Second * 5,
		}

		req, err := http.NewRequest(HttpMethodMapping[proxyReq.HttpMethod], serviceUrl, nil)
		resp, err := netClient.Do(req)
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		internal.CheckError(err)

		err = WriteBytesViaStream(s, bodyBytes)
		internal.CheckError(err)
	})
}

// SetProxyRespStreamHandler set handler for response returned from remote gateway
// TODO: seems not using now
func (n *Node) SetProxyRespStreamHandler() {
	(*n.Host).SetStreamHandler(protocol.ID(ProxyRespStream), func(s network.Stream) {
		buf := bufio.NewReader(s)
		str, err := buf.ReadString('\n')
		if err != nil {
			panic(err)
		}
		fmt.Printf("Read stream: %s\n", str)
	})
}

func (n *Node) NewProxyResp(remoteNode *Node, content []byte) {
	s, err := (*n.Host).NewStream(context.Background(), (*remoteNode.Host).ID(), protocol.ID(ProxyRequestStream))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Stream: %+v\n", s)

	_, err = s.Write(content)
	internal.CheckError(err)
}

func WriteBytesViaStream(s network.Stream, data []byte) error {
	msgLen := len(data)
	msgLenBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(msgLenBytes, uint64(msgLen))

	fmt.Printf("Proxy request len: %+v\n", msgLen)
	_, err := s.Write(msgLenBytes)
	if err != nil {
		return err
	}

	_, err = s.Write(data)
	if err != nil {
		return nil
	}

	return nil
}

func ReadBytesViaStream(s network.Stream) ([]byte, error) {
	reader := bufio.NewReader(s)
	var msgLen uint64
	err := binary.Read(reader, binary.BigEndian, &msgLen)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Received msg len: %+v\n", msgLen)

	proxyReqBuf := make([]byte, msgLen)

	_, err = reader.Read(proxyReqBuf)
	if err != nil {
		return nil, err
	}

	return proxyReqBuf, nil
}
