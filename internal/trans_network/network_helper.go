package trans_network

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"

	"apron.network/gateway-p2p/internal"
	"github.com/libp2p/go-libp2p/core/network"
)

// WriteBytesViaStream writes data byte into network stream.
// It will write the content length (uint64) first to tell the reader how many bytes are followed.
// TODO: Check whether there are method to avoid writing data length first
func WriteBytesViaStream(w io.Writer, data []byte) {
	msgLen := len(data)
	msgLenBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(msgLenBytes, uint64(msgLen))

	switch w.(type) {
	case network.Stream:
		log.Printf("WriteBytesViaStream: data len: %+v, stream: %+v\n", msgLen, w.(network.Stream).Protocol())
	default:
		log.Printf("WriteBytesViaStream: data len: %+v, writer: %+v\n", msgLen, w)
	}
	_, err := w.Write(msgLenBytes)
	internal.CheckError(err)

	writtenSize, err := w.Write(data)
	switch w.(type) {
	case network.Stream:
		log.Printf("WriteBytesViaStream: written %d data to stream: %+v\n", writtenSize, w.(network.Stream).Protocol())
	default:
		log.Printf("WriteBytesViaStream: written %d data to writer: %+v\n", writtenSize, w)
	}
	internal.CheckError(err)
	log.Println("WriteBytesViaStream: Data written")
}

// ReadBytesViaStream reads bytes from network stream. It will read content length first (uint64) then the content bytes.
func ReadBytesViaStream(rd io.Reader, dataCh chan []byte) {
	reader := bufio.NewReader(rd)
	var msgLen uint64
	for {
		err := binary.Read(reader, binary.BigEndian, &msgLen)
		internal.CheckError(err)
		switch rd.(type) {
		case network.Stream:
			log.Printf("ReadBytesViaStream: protocol: %+v, read msg len: %d\n", rd.(network.Stream).Protocol(), msgLen)
		default:
			log.Printf("ReadBytesViaStream: stream: %+v, read msg len: %d\n", rd, msgLen)
		}

		proxyReqBuf := make([]byte, msgLen)

		_, err = reader.Read(proxyReqBuf)
		internal.CheckError(err)

		//log.Printf("ReadBytesViaStream: Received msg from stream: %+v, len: %+v, data: %+q\n", s.Protocol(), msgLen, proxyReqBuf)
		dataCh <- proxyReqBuf
	}
}

//func ReadBytesViaStream(s network.Stream, dataCh chan []byte) {
//	// TODO: Add error chan
//	reader := bufio.NewReader(s)
//	var msgLen uint64
//	for {
//		err := binary.Read(reader, binary.BigEndian, &msgLen)
//		internal.CheckError(err)
//		log.Printf("ReadBytesViaStream: protocol: %+v, read msg len: %d\n", s.Protocol(), msgLen)
//
//		proxyReqBuf := make([]byte, msgLen)
//
//		_, err = reader.Read(proxyReqBuf)
//		internal.CheckError(err)
//
//		//log.Printf("ReadBytesViaStream: Received msg from stream: %+v, len: %+v, data: %+q\n", s.Protocol(), msgLen, proxyReqBuf)
//		dataCh <- proxyReqBuf
//	}
//}
