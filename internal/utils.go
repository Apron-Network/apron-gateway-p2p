package internal

import (
	"bytes"
	"encoding/json"

	"apron.network/gateway-p2p/internal/models"
	"github.com/valyala/fasthttp"
)

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

func ServiceHostnameToIdByte(hostname []byte) []byte {
	return bytes.Replace(hostname, []byte("."), []byte("_"), -1)
}

// RegisterServiceToSSGW is a helper function to register service defined by passed
// in `ApronService` struct to given SSGW address
func RegisterServiceToSSGW(ssgwMgmtAddr string, serviceData models.ApronService) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	newServiceReqBytes, err := json.Marshal(serviceData)
	if err != nil {
		return nil, err
	}

	req.Header.SetRequestURI(ssgwMgmtAddr)
	req.Header.SetMethod("POST")
	req.SetBody(newServiceReqBytes)

	err = fasthttp.Do(req, resp)
	if err != nil {
		return nil, err
	}
	CheckError(err)

	respBytes := resp.Body()
	return respBytes, nil
}

func MergeTwoStruct(oldObj, newObj any) any {
	jsonNew, err := json.Marshal(newObj)
	CheckError(err)

	err = json.Unmarshal(jsonNew, &oldObj)
	CheckError(err)

	return oldObj
}
