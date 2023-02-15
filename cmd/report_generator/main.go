package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"apron.network/gateway-p2p/internal/logger"
	"apron.network/gateway-p2p/internal/models"
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
			logger.GetLogger().Sugar().Infof("Filename: %s, err: %s", path, err)
			return err
		}

		if len(report.Records) != 0 {
			fmt.Printf("Node id: %s", report.NodeId)
			for _, r := range report.Records {
				logger.GetLogger().Sugar().Infof("  Record: %+v", r)
			}
		}

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
