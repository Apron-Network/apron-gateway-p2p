package trans_network

import (
	"github.com/multiformats/go-multiaddr"
	"strings"
)

type AddrList []multiaddr.Multiaddr

func (al *AddrList) String() string {
	strList := make([]string, len(*al))
	for i, addr := range *al {
		strList[i] = addr.String()
	}
	return strings.Join(strList, ",")
}

func (al *AddrList) Set(value string) error {
	addr, err := multiaddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}
