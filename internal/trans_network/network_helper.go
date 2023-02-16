package trans_network

import (
	"bufio"
	"encoding/binary"
	"io"

	"apron.network/gateway-p2p/internal"
	"apron.network/gateway-p2p/internal/logger"
	"github.com/libp2p/go-libp2p/core/network"
	"go.uber.org/zap"
)

// WriteBytesViaStream writes data byte into network stream.
// It will write the content length (uint64) first to tell the reader how many bytes are followed.
func WriteBytesViaStream(w io.Writer, data []byte) {
	msgLen := len(data)
	msgLenBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(msgLenBytes, uint64(msgLen))

	switch w.(type) {
	case network.Stream:
		logger.GetLogger().Debug(
			"WriteBytesViaStream",
			zap.Any("protocol", w.(network.Stream).Protocol()),
			zap.Int("msg_len", msgLen),
		)
	default:
		logger.GetLogger().Debug(
			"WriteBytesViaStream",
			zap.Int("msg_len", msgLen),
		)
	}
	_, err := w.Write(msgLenBytes)
	internal.CheckError(err)

	writtenSize, err := w.Write(data)
	switch w.(type) {
	case network.Stream:
		logger.GetLogger().Debug(
			"WriteBytesViaStream",
			zap.Int("written_size", writtenSize),
			zap.Any("protocol", w.(network.Stream).Protocol()),
		)
	default:
		logger.GetLogger().Debug("WriteBytesViaStream", zap.Int("written_size", writtenSize))
	}
	internal.CheckError(err)
}

// ReadBytesViaStream reads bytes from network stream. It will read content length first (uint64) then the content bytes.
func ReadBytesViaStream(rd io.Reader, dataCh chan []byte, errCh chan error) {
	for {
		dataBuf, err := ReadOneFrameDataFromStream(rd)
		if err != nil {
			logger.GetLogger().Error("ReadOneFrameDataFromStream error", zap.Error(err))
			errCh <- err
		}

		dataCh <- dataBuf
	}
}

// ReadOneFrameDataFromStream reads one frame data (msg length + msg bytes) from stream and return data bytes
func ReadOneFrameDataFromStream(rd io.Reader) ([]byte, error) {
	reader := bufio.NewReader(rd)
	var msgLen uint64
	err := binary.Read(reader, binary.BigEndian, &msgLen)
	if err != nil {
		return nil, err
	}

	switch rd.(type) {
	case network.Stream:
		logger.GetLogger().Debug(
			"ReadBytesViaStream",
			zap.Any("protocol", rd.(network.Stream).Protocol()),
			zap.Uint64("msg_len", msgLen),
		)
	default:
		logger.GetLogger().Debug(
			"ReadBytesViaStream",
			zap.Uint64("msg_len", msgLen),
		)
	}

	dataBuf := make([]byte, msgLen)

	readCnt, err := io.ReadFull(reader, dataBuf)
	if err != nil {
		logger.GetLogger().Error("read data failed", zap.Error(err))
		return nil, err
	}

	if uint64(readCnt) != msgLen {
		logger.GetLogger().Sugar().Errorf("read size not equal, expected size %d, read size %d", msgLen, readCnt)
	}

	return dataBuf, nil
}
