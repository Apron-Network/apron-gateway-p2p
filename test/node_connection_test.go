package test

import (
	"context"
	"gotest.tools/assert"
	"sync"
	"testing"
	"time"
)

var ctx = context.Background()

func TestNodeConnection(t *testing.T) {
	bsNodes, clientNodes := BuildKdhtNetwork(ctx, 1, 1)

	time.Sleep(1 * time.Second)

	assert.Equal(t, (*bsNodes[0].Host).Peerstore().Peers().Len(), 2)
	assert.Equal(t, (*clientNodes[0].Host).Peerstore().Peers().Len(), 2)
}

func TestRequestForward(t *testing.T) {

}
