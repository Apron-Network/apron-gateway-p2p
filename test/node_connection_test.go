package test

import (
	"apron.network/gateway-p2p/internal/models"
	"context"
	"testing"
	"time"
)

var ctx = context.Background()

//func TestNodeConnection(t *testing.T) {
//	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
//
//	time.Sleep(1 * time.Second)
//
//	assert.Equal(t, (*bsNodes[0].Host).Peerstore().Peers().Len(), 2)
//	assert.Equal(t, (*clientNodes[0].Host).Peerstore().Peers().Len(), 2)
//}

func TestRequestForward(t *testing.T) {
	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)
	time.Sleep(1 * time.Second)

	bsNode := bsNodes[0]
	clientNode := clientNodes[0]
	req := &models.ApronServiceRequest{
		HttpMethod: models.ApronServiceRequest_GET,
		ServiceUrl: "ifconfig.co",
		Schema:     "https",
	}

	bsNode.NewProxyRequest(clientNode, req)
	time.Sleep(5 * time.Second)
}
