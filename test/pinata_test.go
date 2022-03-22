package test

import (
	"os"
	"testing"

	"apron.network/gateway-p2p/internal/ipfs_agent"
)

var (
	APIKEY    string
	APISECRET string
)

func init() {
	APIKEY = os.Getenv("PINATA_API_KEY")
	APISECRET = os.Getenv("PINATA_SECRET_API_KEY")
}

func skip(t *testing.T) {
	if APIKEY == "" || APISECRET == "" {
		t.Skip("Skipping testing when without set PINATA_API_KEY and PINATA_SECRET_API_KEY enviornment variables")
	}
}

func TestPinContent(t *testing.T) {
	skip(t)

	content := []byte("Hello, Apron!")
	pinata := ipfs_agent.PinataService{APIKey: APIKEY, APISecret: APISECRET}
	if cid, err := pinata.PinContent(content); err != nil {
		t.Error(err)
	} else {
		t.Log(cid)
	}
}
