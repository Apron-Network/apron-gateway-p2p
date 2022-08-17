package trans_network

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p-core/crypto"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"github.com/fasthttp/websocket"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/proto"
)

type Node struct {
	Host   *host.Host
	Config *NodeConfig

	ps                    *pubsub.PubSub
	broadcastServiceTopic *pubsub.Topic
	serviceBroadcastSub   *pubsub.Subscription
	selfID                peer.ID

	mutex *sync.Mutex
	// Save service has no name but only key. The key for service should be uniq
	services map[string]models.ApronService

	// Save service with name, which may contain multiple service with same name, will be used for LB later
	namedServices map[string][]models.ApronService

	// Mapping of service and peer id, the key for this mapping is service key, and value is peer id, to locate service while receiving forward request
	servicePeerMapping map[string]peer.ID

	// Mapping of request id and client ctx. After receiving request from client side,
	// the client side gateway add uniq requestID to forwarded ApronServiceRequest,
	// and the streamID and the ctx of client will be saved here for later usage
	requestIdChanMapping map[string]chan []byte

	clientWsConns      map[string]*websocket.Conn
	serviceWsConns     map[string]*websocket.Conn
	clientSocketConns  map[string]*net.Conn
	clientHttpDataChan map[string]chan []byte

	serviceUsageRecordManager models.UsageRecordManager
}

func NewNode(ctx context.Context, config *NodeConfig) (*Node, error) {
	// 0.0.0.0 will listen on any interface device.
	// TODO: Make the listening address configurable
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", config.InternalPort))

	var pk crypto.PrivKey
	if config.SecretKey == 0 {
		pk, _, _ = crypto.GenerateEd25519Key(rand.Reader)
	} else {
		pk, _, _ = crypto.GenerateEd25519Key(strings.NewReader(fmt.Sprintf("%032b", config.SecretKey)))
	}

	h, err := libp2p.New(context.Background(), libp2p.ListenAddrs(sourceMultiAddr), libp2p.Identity(pk))
	if err != nil {
		return nil, err
	}

	usageRecordManager := models.UsageRecordManager{}
	usageRecordManager.Init()

	return &Node{
		Host:                      &h,
		Config:                    config,
		services:                  map[string]models.ApronService{},
		namedServices:             map[string][]models.ApronService{},
		servicePeerMapping:        map[string]peer.ID{},
		mutex:                     &sync.Mutex{},
		requestIdChanMapping:      map[string]chan []byte{},
		clientWsConns:             map[string]*websocket.Conn{},
		serviceWsConns:            map[string]*websocket.Conn{},
		clientHttpDataChan:        map[string]chan []byte{},
		serviceUsageRecordManager: usageRecordManager,
	}, nil
}

/*******************************/
// Self connection related logic
/*******************************/

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

	log.Printf("[Remote Service] Broadcast Listener started")

	go n.StartListeningOnServiceBroadcast(ctx)
}

// StartListeningOnServiceBroadcast is a infinity loop which listens to service broadcast subscriptions.
// After receiving the message, the service will be added to remote service list of this node,
// which will be queried while receiving service request
func (n *Node) StartListeningOnServiceBroadcast(ctx context.Context) {
	for {
		msg, err := n.serviceBroadcastSub.Next(ctx)
		if err != nil {
			log.Println("wait ServiceBroadcast err", err)
			continue
		}

		// log.Printf("ReceivedFrom: %+s\n", msg.ReceivedFrom.Pretty())

		if msg.ReceivedFrom == n.selfID {
			continue
		}
		service := new(models.ApronService)
		err = proto.Unmarshal(msg.Data, service)
		if err != nil {
			log.Println("Unmarshal ApronService err", err)
			continue
		}
		// log.Printf("Received service: %+v\n", service)
		n.RegisterRemoteService(msg.ReceivedFrom, service)
	}
}

// BroadcastService broad local service to the network with configured topic,
// so all nodes subscribed to the topic can update its local cache data
func (n *Node) BroadcastService(ctx context.Context, service *models.ApronService) error {
	log.Printf("[Local Service] broadcast service %s --to--> all peers\n", service.Id)
	data, err := proto.Marshal(service)
	if err != nil {
		return err
	}
	return n.broadcastServiceTopic.Publish(ctx, data)
}

func (n *Node) RegisterLocalService(service *models.ApronService) {
	log.Printf("\n")
	n.mutex.Lock()
	if service.IsDeleted {
		log.Printf("[Local Service] deleted service %s\n", service.Id)
		delete(n.services, service.Id)
		delete(n.servicePeerMapping, service.Id)
	} else {
		n.services[service.Id] = *service
		n.servicePeerMapping[service.Id] = n.selfID
		log.Printf("[Local Service] new/update service: %s\n", service.Id)
		log.Printf("[Local Service] new/update service %+v \n", service)
	}

	n.mutex.Unlock()
	if err := n.BroadcastService(context.Background(), service); err != nil {
		log.Println("RegisterLocalService err", err)
		panic(err)
	}
}

func (n *Node) RegisterRemoteService(peerId peer.ID, service *models.ApronService) {
	log.Printf("\n")
	log.Printf("[Remote Service] from remote(%s) -to-> local(%s)\n", peerId.String(), n.selfID.String())
	n.mutex.Lock()

	if service.IsDeleted {
		log.Printf("[Remote Service] deleted service %s\n", service.Id)
		delete(n.services, service.Id)
		delete(n.servicePeerMapping, service.Id)
	} else {
		n.services[service.Id] = *service
		n.servicePeerMapping[service.Id] = peerId
		log.Printf("[Remote Service] new/update service %+v \n", service)
	}

	n.mutex.Unlock()
	log.Printf("\n")
}

func (n *Node) NodeAddrStr() string {
	bsIdStr := (*n.Host).ID().Pretty()
	bsNodeAddr := (*n.Host).Addrs()[0]
	return fmt.Sprintf("%s/p2p/%s", bsNodeAddr, bsIdStr)
}

// UpdatePeers : if a peer disconnected and wasn't found in topic, all services related to it will be removed.
func (n *Node) UpdatePeers() {
	peerRefreshTicker := time.NewTicker(time.Second)
	defer peerRefreshTicker.Stop()
	log.Printf("[Remote Service] Peers Monitor started")
	for {
		<-peerRefreshTicker.C
		availablePeers := n.ps.ListPeers(BroadcastServiceChannel)
		// log.Printf("availablePeers %++v", availablePeers)
		invaildService := make([]string, 0)
		n.mutex.Lock()
		for k, v := range n.servicePeerMapping {
			if v == n.selfID {
				continue
			}
			found := false
			for _, p := range availablePeers {
				if v == p {
					found = true
				}
			}

			if !found {
				invaildService = append(invaildService, k)
			}

		}

		// remove related services
		for _, service := range invaildService {
			log.Printf("[Remote Service] peer %s disconnected\n", n.servicePeerMapping[service])
			log.Printf("[Remote Service] remove service %s\n", n.services[service].Id)
			delete(n.services, service)
			delete(n.servicePeerMapping, service)
		}
		n.mutex.Unlock()

	}
}
