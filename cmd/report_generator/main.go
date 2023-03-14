package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"apron.network/gateway-p2p/internal/logger"
	"apron.network/gateway-p2p/internal/models"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func main() {
	apiKey := flag.String("ipfs-key", "", "Api key for IPFS agent")
	apiSecret := flag.String("ipfs-secret", "", "Api secret for IPFS agent")
	flag.Parse()

	baseDir := fmt.Sprintf("/tmp/%s/%s", *apiKey, *apiSecret)
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if err != nil {
			fmt.Println(err)
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Println(err)
			return err
		}

		var report models.NodeReport
		err = proto.Unmarshal(content, &report)
		if err != nil {
			logger.GetLogger().Error("unmarshal content error", zap.String("filename", path), zap.Error(err))
			return err
		}

		if len(report.Records) != 0 {
			logger.GetLogger().Debug("report info", zap.String("node_id", report.NodeId), zap.Any("records", report.Records))
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
