package main

import (
	"bufio"
	"flag"
	"net"

	"go.uber.org/zap"
)

var logger = zap.NewExample().Sugar()

func main() {
	listenAddr := flag.String("listen-addr", ":10800", "Listen address for socket server")
	flag.Parse()

	l, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		panic(err)
	}

	logger.Info("##########################################")
	logger.Info("Test Socket Server, only echo message back")
	logger.Info("The message delimiter is *\n* ")
	logger.Info("##########################################")
	logger.Infof("Start listening on %s ...\n", *listenAddr)

	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go serveConnection(conn)
	}

}

func serveConnection(conn net.Conn) {
	defer conn.Close()
	logger.Infof("serve connection from %s", conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)

	for {
		bytes, err := reader.ReadBytes(byte('\n'))
		if err != nil {
			panic(err)
		}

		logger.Infof("Received %q", bytes)
		conn.Write(bytes)
	}
}
