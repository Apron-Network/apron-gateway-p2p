package ipfs_agent

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	fp "path/filepath"
	"time"
)

type PinataService struct {
	APIKey    string
	APISecret string
}

const (
	PIN_FILE_ENDPOINT = "https://api.pinata.cloud/pinning/pinFileToIPFS"
)

func (p *PinataService) PinFile(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		defer w.Close()
		defer m.Close()

		part, err := m.CreateFormFile("file", fp.Base(file.Name()))
		if err != nil {
			return
		}

		if _, err = io.Copy(part, file); err != nil {
			return
		}
	}()

	req, err := http.NewRequest(http.MethodPost, PIN_FILE_ENDPOINT, r)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", m.FormDataContentType())
	req.Header.Add("pinata_secret_api_key", p.APISecret)
	req.Header.Add("pinata_api_key", p.APIKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var json_data map[string]interface{}
	if err := json.Unmarshal(data, &json_data); err != nil {
		return "", err
	}

	if out, err := json_data["error"].(string); err {
		return "", fmt.Errorf(out)
	}
	if hash, ok := json_data["IpfsHash"].(string); ok {
		return hash, nil
	}

	return "", fmt.Errorf("pinata pin file failure")
}
