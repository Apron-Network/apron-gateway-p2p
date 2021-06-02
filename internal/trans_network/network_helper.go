package trans_network

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
)

// WriteBytesViaStream writes data byte into network stream.
// It will write the content length (uint64) first to tell the reader how many bytes are followed.
// TODO: Check whether there are method to avoid writing data length first
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

// ReadBytesViaStream reads bytes from network stream. It will read content length first (uint64) then the content bytes.
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
