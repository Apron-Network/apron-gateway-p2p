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
	for {
		dataBuf, err := ReadOneFrameDataFromStream(rd)
		internal.CheckError(err)

		//log.Printf("ReadBytesViaStream: Received msg from stream: %+v, len: %+v, data: %+q\n", s.Protocol(), msgLen, proxyReqBuf)
		dataCh <- dataBuf
	}
}

// ReadOneFrameDataFromStream reads one frame data (msg length + msg bytes) from stream and return data bytes
func ReadOneFrameDataFromStream(rd io.Reader) ([]byte, error) {
	reader := bufio.NewReader(rd)
	var msgLen uint64
	err := binary.Read(reader, binary.BigEndian, &msgLen)
	internal.CheckError(err)
	log.Printf("frame data size: %d", msgLen)
	switch rd.(type) {
	case network.Stream:
		log.Printf("ReadBytesViaStream: protocol: %+v, read msg len: %d\n", rd.(network.Stream).Protocol(), msgLen)
	default:
		log.Printf("ReadBytesViaStream: stream: %+v, read msg len: %d\n", rd, msgLen)
	}

	dataBuf := make([]byte, msgLen)

	_, err = reader.Read(dataBuf)
	if err != nil {
		// TODO: Add zap logger
		return nil, err
	}

	return dataBuf, nil
}
