package trans_network

import (
	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/models"
	"bufio"
	"encoding/binary"
	"github.com/fasthttp/websocket"
	"github.com/libp2p/go-libp2p-core/network"
	"google.golang.org/protobuf/proto"
	"log"
)

// WriteBytesViaStream writes data byte into network stream.
// It will write the content length (uint64) first to tell the reader how many bytes are followed.
// TODO: Check whether there are method to avoid writing data length first
func WriteBytesViaStream(s network.Stream, data []byte) error {
	msgLen := len(data)
	msgLenBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(msgLenBytes, uint64(msgLen))

	log.Printf("WriteBytesViaStream: Proxy request len: %+v, stream: %+v, data: %+q\n", msgLen, s.ID(), data)
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

	log.Printf("Received msg len: %+v\n", msgLen)

	proxyReqBuf := make([]byte, msgLen)

	_, err = reader.Read(proxyReqBuf)
	if err != nil {
		return nil, err
	}

	return proxyReqBuf, nil
}

func ParseProxyReqFromStream(s network.Stream) (*models.ApronServiceRequest, error) {
	msgByte, err := ReadBytesViaStream(s)
	if err != nil {
		return nil, err
	}
	internal.CheckError(err)

	proxyReq := &models.ApronServiceRequest{}
	err = proto.Unmarshal(msgByte, proxyReq)
	if err != nil {
		return nil, err
	}

	return proxyReq, nil
}

func ParseProxyDataFromStream(s network.Stream) (*models.ApronServiceData, error) {
	msgByte, err := ReadBytesViaStream(s)
	if err != nil {
		return nil, err
	}
	internal.CheckError(err)

	proxyData := &models.ApronServiceData{}
	err = proto.Unmarshal(msgByte, proxyData)
	if err != nil {
		return nil, err
	}

	return proxyData, nil
}

func ForwardWsMsgToInternalStream(src *websocket.Conn, dest *network.Stream, errCh chan error) {
	for {
		msgType, msgBytes, err := src.ReadMessage()
		if err != nil {
			log.Printf("src.ReadMessage failed, msgType=%d, msg=%s, err=%v\n", msgType, msgBytes, err)
			if ce, ok := err.(*websocket.CloseError); ok {
				msgBytes = websocket.FormatCloseMessage(ce.Code, ce.Text)
			} else {
				msgBytes = websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, err.Error())
			}

			errCh <- err

			// TODO: Write close to stream

			break
		}

		err = WriteBytesViaStream(*dest, msgBytes)
		if err != nil {
			log.Printf("dest.WriteMessage error: %v\n", err)
			errCh <- err
			break
		}
	}
}

func ForwardInternalStreamToWsMsg(src *network.Stream, dest *websocket.Conn, errCh chan error) {
	for {
		msgByte, err := ReadBytesViaStream(*src)
		if err != nil {
			log.Printf("Read stream message error: %v\n", err)
			errCh <- err
			break
		}

		err = dest.WriteMessage(websocket.TextMessage, msgByte)
		if err != nil {
			log.Printf("Write ws message error: %v\n", err)
			errCh <- err
			break
		}
	}
}
