package ipfs_agent

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"apron.network/gateway-p2p/internal/logger"
)

// LocalFileAgent uses local file to test IPFS related functions
type LocalFileAgent struct {
	APIKey    string
	APISecret string

	storageBase string
}

func (agent *LocalFileAgent) prepareStorage() {
	if agent.APIKey == "" {
		agent.APIKey = "agTestKey"
	}

	if agent.APISecret == "" {
		agent.APISecret = "agTestSecret"
	}

	agent.storageBase = fmt.Sprintf("/tmp/%s/%s", agent.APIKey, agent.APISecret)

	err := os.MkdirAll(agent.storageBase, 0755)
	if err != nil {
		panic(err)
	}
}

func (agent *LocalFileAgent) PinFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		log.Fatal(err)
	}

	return agent.PinContent(data)
}

func (agent *LocalFileAgent) PinContent(data []byte) (string, error) {
	agent.prepareStorage()

	targetFile, err := ioutil.TempFile(agent.storageBase, "apron-")
	if err != nil {
		log.Fatal(err)
	}
	defer targetFile.Close()

	bytesWritten, err := targetFile.Write(data)
	if err != nil {
		log.Fatal(err)
	}

	logger.GetLogger().Sugar().Infof("Bytes Written: %d", bytesWritten)
	return targetFile.Name(), nil
}
