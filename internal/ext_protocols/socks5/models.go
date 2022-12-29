package socks5

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
)

// AuthContext is a Request encapsulates authentication state provided during negotiation
type AuthContext struct {
	// Provided auth method
	Method uint8
	// Payload provided during negotiation.
	// Keys depend on the used auth method.
	// For UserPassauth contains Username
	Payload map[string]string
}

// AddrSpec is used to return the target AddrSpec
// which may be specified as IPv4, IPv6, or a FQDN
type AddrSpec struct {
	FQDN string
	IP   net.IP
	Port int
}

// A Request represents request received by a server
type Request struct {
	Version      uint8
	Command      uint8
	AuthContext  *AuthContext
	RemoteAddr   *AddrSpec
	DestAddr     *AddrSpec
	realDestAddr *AddrSpec
	bufConn      io.Reader
}

func (r *Request) toBytes() []byte {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(r)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	return b.Bytes()
}

func GetRequestFromPack(dataPack []byte) Request {
	r := Request{}
	b := bytes.Buffer{}
	b.Write(dataPack)
	d := gob.NewDecoder(&b)
	err := d.Decode(&r)
	if err != nil {
		fmt.Println(`failed gob Decode`, err)
	}
	return r
}
