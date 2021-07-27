package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"bufio"
	"encoding/binary"
	"github.com/libp2p/go-libp2p-core/network"
	"log"
)

// WriteBytesViaStream writes data byte into network stream.
// It will write the content length (uint64) first to tell the reader how many bytes are followed.
// TODO: Check whether there are method to avoid writing data length first
func WriteBytesViaStream(s network.Stream, data []byte) {
	msgLen := len(data)
	msgLenBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(msgLenBytes, uint64(msgLen))

	log.Printf("WriteBytesViaStream: data len: %+v, stream: %+v, data: %+q\n", msgLen, s.Protocol(), data)
	_, err := s.Write(msgLenBytes)
	internal.CheckError(err)

	writtenSize, err := s.Write(data)
	log.Printf("WriteBytesViaStream: written %d data to stream: %+v\n", writtenSize, s.Protocol())
	internal.CheckError(err)
	log.Println("WriteBytesViaStream: Data written")
}

// ReadBytesViaStream reads bytes from network stream. It will read content length first (uint64) then the content bytes.
func ReadBytesViaStream(s network.Stream, dataCh chan []byte) {
	// TODO: Add error chan
	reader := bufio.NewReader(s)
	var msgLen uint64
	for {
		err := binary.Read(reader, binary.BigEndian, &msgLen)
		internal.CheckError(err)
		log.Printf("ReadBytesViaStream: protocol: %+v, read msg len: %d\n", s.Protocol(), msgLen)

		proxyReqBuf := make([]byte, msgLen)

		_, err = reader.Read(proxyReqBuf)
		internal.CheckError(err)

		log.Printf("ReadBytesViaStream: Received msg from stream: %+v, len: %+v, data: %+q\n", s.Protocol(), msgLen, proxyReqBuf)
		dataCh <- proxyReqBuf
	}
}
