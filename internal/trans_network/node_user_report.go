package trans_network

import (
	"apron.network/gateway-p2p/internal/ipfs_agent"
	"fmt"
	"google.golang.org/protobuf/proto"
	"log"
	"time"
)

func (n *Node) StartUploadUsageReportTask(uploadInterval int, ipfsAgent ipfs_agent.IpfsAgent) {
	for true {
		if nodeReport, err := n.serviceUsageRecordManager.ExportAllUsage(n.selfID.String()); err != nil {
			fmt.Printf(fmt.Errorf("error occurred while exporting usage report: %+v", err).Error())
		} else {
			if len(nodeReport.Records) != 0 {
				// Only export records has real usage data
				reportBytes, err := proto.Marshal(&nodeReport)
				if err != nil {
					// TODO: replace with appropriate handler
					panic(err)
				}
				fileHash, err := ipfsAgent.PinContent(reportBytes)
				if err != nil {
					log.Fatal(err)
				}
				// TODO: Upload the file hash to chain
				fmt.Printf("Filehash: %s\n", fileHash)
			}
		}
		time.Sleep(time.Duration(uploadInterval) * time.Second)
	}

}
